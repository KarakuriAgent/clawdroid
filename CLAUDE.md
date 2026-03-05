# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

ClawDroid は Go バックエンドを内蔵した Android AI アシスタント。Go シングルバイナリ（エージェントループ・ツール実行・LLM 呼び出し・メッセージングチャンネル）と Kotlin/Jetpack Compose フロントエンド（チャット UI・音声モード・デバイス自動操作）で構成される。

## 作業ルール

コードを修正する前に、必ずプランモードでプランを作成し、ユーザーの承認を得てから実装に入ること。
PR 作成時は最新の main ブランチから新しいブランチを作成し、`.github/pull_request_template.md` のテンプレートに従うこと。
Issue 作成時は `.github/ISSUE_TEMPLATE/` のテンプレートに従うこと。

## ビルド・テストコマンド

### Go バックエンド
```bash
make build          # 現在のプラットフォーム向けにビルド
make build-all      # linux/{amd64,arm64,arm} 向けに全ビルド
make test           # go test ./... を実行
make check          # deps + fmt + vet + test をまとめて実行
make vet            # 静的解析
make fmt            # コードフォーマット
make run            # ビルド後に実行
```

単一パッケージのテスト: `go test ./pkg/tools/...`

### Android アプリ
```bash
make build-android              # Go バックエンドを jniLibs としてビルド（全アーキテクチャ）
make build-android-arm64        # arm64-v8a のみ
cd android && ./gradlew assembleEmbeddedDebug   # Embedded フレーバー（Go バックエンド込み）
cd android && ./gradlew assembleTermuxDebug      # Termux フレーバー（バックエンドなし）
```

## 通信フロー

Android アプリ ↔ Go バックエンド: WebSocket (`ws://127.0.0.1:18793`)
設定 API: HTTP Gateway (`127.0.0.1:18790`)

## 設定

設定ファイル: `~/.clawdroid/config.json`
環境変数: `CLAWDROID_*` プレフィックスで上書き可能（例: `CLAWDROID_LLM_API_KEY`）
バージョン管理: `/VERSION` ファイルにバージョン文字列を格納

## GitHub テンプレート・CI

### Issue テンプレート (`.github/ISSUE_TEMPLATE/`)
- **Bug report** (`[BUG]`ラベル): 環境情報（ClawDroid バージョン、Go バージョン、AI モデル、OS、チャンネル）＋再現手順
- **Feature request** (`[Feature]`ラベル): ゴール、提案、影響度（Core Feature / Nice-to-Have / Roadmap 連携）
- **General Task** (`[Task]`ラベル): 目的、ToDoリスト、完了条件

### PR テンプレート (`.github/pull_request_template.md`)
- Type of Change: Bug fix / New feature / Documentation / Refactoring から選択
- Linked Issue、Technical Context、Test Environment（ハードウェア・OS・モデル・チャンネル）、Proof of Work を記載

### CI ワークフロー (`.github/workflows/`)
- `pr.yml` / `go-build.yml`: Go ビルド・テスト（push/PR 時）
- `android-pr.yml` / `android-build.yml`: Android APK ビルド
- `release.yml`: GoReleaser によるリリース自動化
- `version-bump.yml`: バージョン番号の自動更新 PR
