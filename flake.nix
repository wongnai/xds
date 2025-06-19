{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-25.05";
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, gitignore }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; };
      stdenv = pkgs.stdenv;
      lib = pkgs.lib;
      source = gitignore.lib.gitignoreSource self;
    in rec {
      packages = {
        default = pkgs.buildGoModule {
          name = "xds";
          src = gitignore.lib.gitignoreSource ./.;
          # To update this, set the value to `lib.fakeHash` then build with `nix build '.#'`
          # Nix will tell you the correct value.
          vendorHash = "sha256-1/5X4eNf/nKiRxjcmNu7AbU1KYn/MppCdEH671TclLE=";
        };
        container = pkgs.dockerTools.buildLayeredImage {
          name = "xds";
          contents = with pkgs; [ dockerTools.caCertificates ];
          config.Cmd = [ "${packages.default}/bin/xds" ];
        };
      };
      devShells = {
        default = pkgs.mkShell {
          packages = with pkgs; [ go ];
        };
      };
    }
  );
}
