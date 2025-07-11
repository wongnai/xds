package snapshot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

const AnnotationRetryableStatusCode = "xds.lmwn.com/retry-status"
const AnnotationNumRetries = "xds.lmwn.com/retry-count"
const AnnotationRetryBackoff = "xds.lmwn.com/retry-backoff"

// retryPolicyFromService convert annotations on the Kubernetes service to xDS retry policy
// It may return nil if not configured
//
// The supported values are as in gRPC [A44](https://github.com/grpc/proposal/blob/master/A44-xds-retry.md)
func retryPolicyFromService(service *corev1.Service) *routev3.RetryPolicy {
	annotations := service.GetAnnotations()
	retryableStatusCode, ok := annotations[AnnotationRetryableStatusCode]
	if !ok {
		return nil
	}

	out := &routev3.RetryPolicy{
		RetryOn:      retryableStatusCode,
		RetryBackOff: nil,
	}

	numRetries, ok := annotations[AnnotationNumRetries]
	if ok {
		parsed, err := strconv.ParseUint(numRetries, 10, 32)
		if err == nil {
			out.NumRetries = wrapperspb.UInt32(uint32(parsed))
		} else {
			klog.ErrorS(err, "cannot parse num-retries", "object", klog.KObj(service), "num-retries", numRetries)
		}
	}

	retryBackoff, ok := annotations[AnnotationRetryBackoff]
	if ok {
		retryPolicy, err := parseRetryPolicyBackoff(retryBackoff)
		if err == nil {
			out.RetryBackOff = retryPolicy
		} else {
			klog.ErrorS(err, "cannot parse retry-backoff", "object", klog.KObj(service), "num-retries", numRetries)
		}
	}

	return out
}

func parseRetryPolicyBackoff(v string) (*routev3.RetryPolicy_RetryBackOff, error) {
	parts := strings.SplitN(v, ",", 2)
	switch len(parts) {
	case 0:
		return nil, nil
	case 1:
		durs, err := time.ParseDuration(v)
		if err != nil {
			return nil, err
		}
		return &routev3.RetryPolicy_RetryBackOff{
			BaseInterval: durationpb.New(durs),
		}, nil
	case 2:
		durs1, err := time.ParseDuration(parts[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse first duration: %w", err)
		}
		durs2, err := time.ParseDuration(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse second duration: %w", err)
		}
		return &routev3.RetryPolicy_RetryBackOff{
			BaseInterval: durationpb.New(durs1),
			MaxInterval:  durationpb.New(durs2),
		}, nil
	default:
		panic("unexpected split!")
	}
}
