{
  description = "open-atlas-search golang nixos flake";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/master";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        system = "x86_64-linux";
        config.allowUnfree = true;
      };
    in {
      devShell = pkgs.mkShell {
        nativeBuildInputs = [ pkgs.bashInteractive ];
        buildInputs = with pkgs; [
          go
          nodejs_22
          yarn
        ];
        shellHook = with pkgs; ''
          # fixes libstdc++ issues and libgl.so issues
          export LD_LIBRARY_PATH=${lib.makeLibraryPath [stdenv.cc.cc]}
        '';
      };
    });
}
