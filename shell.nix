{ pkgs ? (import <nixpkgs> {}) }:
pkgs.mkShell {
  buildInputs = [
    pkgs.elmPackages.elm
    pkgs.elmPackages.elm-format
  ];
}
