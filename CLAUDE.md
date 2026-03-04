# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

ClawDroid は Go バックエンドを内蔵した Android AI アシスタント。Go シングルバイナリ（エージェントループ・ツール実行・LLM 呼び出し・メッセージングチャンネル）と Kotlin/Jetpack Compose フロントエンド（チャット UI・音声モード・デバイス自動操作）で構成される。

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

## 言語ルール

コード上のコメント、ドキュメント、コミットメッセージ、Issue、PR はすべて英語で書くこと。ユーザーから別途指示がある場合のみ他言語を使用する。

## PR・Issue 作成ルール

PR 作成時は `.github/pull_request_template.md` のフォーマットに従うこと:
- **Description**: 変更内容の説明
- **Type of Change**: Bug fix / New feature / Documentation / Refactoring から該当するものをチェック
- **Linked Issue**: 関連 Issue があればリンク
- **Technical Context**: ドキュメント以外の変更時は参考 URL・理由を記載
- **Test Environment & Hardware**: テストした環境（Hardware, OS, Model/Provider, Channels）
- **Checklist**: スタイル準拠・セルフレビュー・ドキュメント更新の確認

Issue 作成時は `.github/ISSUE_TEMPLATE/` のテンプレートを使用:
- **Bug report** (`[BUG]`): 環境情報＋再現手順
- **Feature request** (`[Feature]`): ゴール、提案、影響度
- **General Task** (`[Task]`): 目的、ToDoリスト、完了条件

## CI ワークフロー (`.github/workflows/`)

- `pr.yml` / `go-build.yml`: Go ビルド・テスト（push/PR 時）
- `android-pr.yml` / `android-build.yml`: Android APK ビルド
- `release.yml`: GoReleaser によるリリース自動化
- `version-bump.yml`: バージョン番号の自動更新 PR
