{
  description = "gothink";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "gothink";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-h4td20xNs2a5Hats8dIhzpZPHDc0/rxEIM98/f9XbHY=";
          meta.mainProgram = "gothink";
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/gothink";
        };
      }
    );
}
