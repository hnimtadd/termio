{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  # Optimize shell startup by reducing shellHook complexity
  shellHook = ''
  '';

  # Group related packages for better caching
  nativeBuildInputs = with pkgs.buildPackages; [
    # Go toolchain
    go_1_24
    gopls
    delve
    golangci-lint
    gotools
    go-mockery

    # AI/ML tools
    nodejs-slim

    # Development essentials
    git
    gnumake
    direnv
    fd
    ripgrep
    uutils-coreutils-noprefix
  ];
}
