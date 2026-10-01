{
  description = "Tools for brainium-style org-mode knowledge repositories";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    {
      self,
      nixpkgs,
    }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      commands = [
        "orglint"
        "notesearch"
        "notestatus"
        "notesearch-eval"
      ];
    in
    {
      packages = forAllSystems (
        pkgs:
        let
          # subPackages を指定しないので cmd が全部 1 つの derivation に入る。
          # cmd ごとに分けると checkPhase の go test もその範囲に縮み、cmd/
          # にテストが無いぶん internal/ のテストが nix flake check から丸ご
          # と外れるため、分けずに束ねている
          brainium-tools = pkgs.buildGoLatestModule {
            pname = "brainium-tools";
            version = "0.1.0";
            src = self;
            vendorHash = null;
          };
        in
        {
          inherit brainium-tools;
          default = brainium-tools;
        }
      );

      # 1 つの derivation に束ねたコマンドを nix run .#orglint のように
      # 名前で呼べるようにする
      apps = forAllSystems (
        pkgs:
        let
          brainium-tools = self.packages.${pkgs.stdenv.hostPlatform.system}.brainium-tools;
        in
        nixpkgs.lib.genAttrs commands (name: {
          type = "app";
          program = "${brainium-tools}/bin/${name}";
          meta.description = "${name} from brainium-tools";
        })
      );

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go_latest
            pkgs.gopls
          ];
        };
      });

      checks = forAllSystems (pkgs: {
        # buildGoLatestModule の checkPhase が go test ./... を実行する
        inherit (self.packages.${pkgs.stdenv.hostPlatform.system}) brainium-tools;
      });

      formatter = forAllSystems (
        pkgs:
        pkgs.nixfmt-tree.override {
          runtimeInputs = [ pkgs.go_latest ];
          settings.formatter.gofmt = {
            command = "gofmt";
            options = [ "-w" ];
            includes = [ "*.go" ];
          };
        }
      );
    };
}
