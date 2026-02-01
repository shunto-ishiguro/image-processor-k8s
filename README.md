# Kubernetes ハンズオン学習

画像処理APIを使って、Kubernetesの価値を**体験**するプロジェクト。

---

## このプロジェクトの全体像

### 何を作るの？

「画像を送ったら、加工して返す」APIサーバー。

```
┌──────────┐    画像を送る     ┌──────────┐
│  ユーザー │ ───────────────→ │  サーバー │
│          │ ←─────────────── │ (Go API) │
└──────────┘   加工された画像   └──────────┘
```

例：
- 大きい画像を送る → 小さくリサイズして返す
- カラー画像を送る → 白黒にして返す
- 画像を送る → 90度回転して返す

### Kubernetesで何を学ぶの？

このAPIサーバーを**Kubernetes上で動かす**ことで：

1. **自動復旧** - サーバーが落ちても自動で復活
2. **自動スケーリング** - アクセスが増えたら自動でサーバーを増やす
3. **ゼロダウンタイム更新** - サービスを止めずにバージョンアップ

を体験する。

### ファイル構成

```
image-processing-k8s/
├── cmd/api/main.go        # サーバーのエントリーポイント
├── internal/              # 画像処理のロジック
├── testdata/              # テスト用の画像（自動生成）
├── output/                # 処理結果の保存先
├── k8s/                   # Kubernetesの設定ファイル
├── scripts/               # 便利スクリプト
├── docs/                  # ドキュメント
├── Dockerfile             # Dockerイメージの作り方
└── Makefile               # よく使うコマンド集
```

### ドキュメント

- **[docs/basics.md](docs/basics.md)** - Kubernetes用語集・基礎知識
- **[docs/k8s-manifests.md](docs/k8s-manifests.md)** - k8s/フォルダのYAMLファイル解説

---

## なぜKubernetesが必要？

普通にサーバーを動かすだけなら `go run main.go` で十分。

でも本番環境では：
- アクセスが急増したら？ → **自動でサーバーを増やしたい**
- サーバーが落ちたら？ → **自動で復旧してほしい**
- 新バージョンをデプロイしたい → **サービスを止めずに更新したい**

これを実現するのがKubernetes。

このハンズオンで、実際にそれを体験する。

---

## 準備

必要なもの：
- Go 1.22+
- Docker
- minikube（または kind、Docker Desktop の Kubernetes）
- kubectl

```bash
# 確認
go version
docker --version
minikube version
kubectl version --client
```

---

## Step 0: まず画像処理APIを動かす

Kubernetesなしでも普通に動くことを確認。

### テスト画像を生成
```bash
make generate-image
# → testdata/sample.jpg が作られる（800x600のカラフルな画像）
```

### サーバー起動
```bash
make run
# または: go run ./cmd/api
```

### 画像処理を試す（別ターミナル）
```bash
# リサイズ（800x600 → 200x150）
curl -X POST -F "image=@testdata/sample.jpg" \
  "http://localhost:8080/api/resize?width=200&height=150" \
  -o output/resized.jpg

# グレースケール変換
curl -X POST -F "image=@testdata/sample.jpg" \
  "http://localhost:8080/api/grayscale" \
  -o output/gray.jpg

# 90度回転
curl -X POST -F "image=@testdata/sample.jpg" \
  "http://localhost:8080/api/rotate?angle=90" \
  -o output/rotated.jpg
```

```bash
# 出力確認
mkdir -p output
ls -la output/
```

動いた。でもこれだと：
- サーバー1台だけ
- 落ちたら終わり
- 負荷が増えても1台で頑張るしかない

---

## Step 1: Kubernetesクラスターを起動

```bash
# minikubeを起動
minikube start

# 確認
kubectl cluster-info
```

---

## Step 2: Dockerイメージを作る

```bash
# minikubeのDockerを使う（重要！）
eval $(minikube docker-env)

# イメージをビルド
docker build -t image-api:latest .

# 確認
docker images | grep image-api
```

> **なぜ `eval $(minikube docker-env)` が必要？**
>
> minikubeは独自のDockerを持っている。
> これをしないと、minikubeからイメージが見えない。

---

## Step 3: Kubernetesにデプロイ

```bash
# まとめてデプロイ
make k8s-apply

# 状態確認
kubectl get all -n image-api
```

出力例：
```
NAME                             READY   STATUS    RESTARTS   AGE
pod/image-api-xxxxx-aaaaa        1/1     Running   0          30s
pod/image-api-xxxxx-bbbbb        1/1     Running   0          30s

NAME                        TYPE        CLUSTER-IP      PORT(S)
service/image-api           ClusterIP   10.96.xxx.xxx   80/TCP
service/image-api-nodeport  NodePort    10.96.xxx.xxx   80:30080/TCP

NAME                        READY   UP-TO-DATE   AVAILABLE
deployment.apps/image-api   2/2     2            2
```

**Podが2つ動いている** = 画像処理サーバーが2台起動した

---

## Step 4: Kubernetes上で画像処理を試す

```bash
# ポートフォワード（Kubernetes内のサービスにアクセスする方法）
kubectl port-forward service/image-api 8080:80 -n image-api
```

別ターミナルで：
```bash
# 画像をリサイズ
curl -X POST -F "image=@testdata/sample.jpg" \
  "http://localhost:8080/api/resize?width=100" \
  -o output/k8s-resized.jpg

# 確認
ls -la output/k8s-resized.jpg
```

Kubernetes上のサーバーで画像処理ができた。

---

## Step 5: 【体験】自己修復を見る

Kubernetesの価値その1：**サーバーが落ちても自動復旧**

### ターミナル1: Podを監視
```bash
kubectl get pods -n image-api -w
```

### ターミナル2: Podを殺す
```bash
# Pod名を確認
kubectl get pods -n image-api

# 1つ削除してみる（サーバーを強制終了するイメージ）
kubectl delete pod <Pod名> -n image-api
```

### 観察結果

ターミナル1を見ると：
```
NAME                         READY   STATUS
image-api-xxxxx-aaaaa        1/1     Running
image-api-xxxxx-bbbbb        1/1     Running
image-api-xxxxx-aaaaa        1/1     Terminating  ← 削除された
image-api-xxxxx-ccccc        0/1     Pending      ← 新しいPodが作られた
image-api-xxxxx-ccccc        1/1     Running      ← 復旧完了
```

**サーバーを殺しても、自動で新しいサーバーが立ち上がる。**

---

## Step 6: 【体験】手動スケーリング

Kubernetesの価値その2：**簡単にサーバー台数を変更**

```bash
# 現在のPod数を確認
kubectl get pods -n image-api
# → 2台

# 5台に増やす
kubectl scale deployment image-api --replicas=5 -n image-api

# 確認
kubectl get pods -n image-api
# → 5台に増えた！

# 1台に減らす
kubectl scale deployment image-api --replicas=1 -n image-api
```

コマンド1つでサーバー台数を変更できる。
画像処理のリクエストが増えたら、サーバーを増やせばいい。

---

## Step 7: 【体験】自動スケーリング (HPA)

負荷に応じて**自動で**サーバー台数を増減させる。

### 負荷テストとは？

本番環境では、多くのユーザーが同時にアクセスしてくる：

```
ユーザーA → 「この画像リサイズして」 ─┐
ユーザーB → 「この画像を白黒に」   ─┼─→ サーバー（処理が大変！）
ユーザーC → 「この画像回転して」   ─┤
    :                              │
ユーザー100人が同時にリクエスト ───┘
```

これをシミュレートするのが「負荷テスト」。

### 負荷テストの仕組み

**テスト画像は1枚だけ**でいい。同じ画像を何度も送って「大量のユーザー」を再現する。

```
┌─────────────────────────────────────────────┐
│ 負荷テストスクリプト                          │
│                                             │
│  sample.jpg を使って:                        │
│  ├── リクエスト1: リサイズして → サーバー     │
│  ├── リクエスト2: グレースケールにして        │
│  ├── リクエスト3: 回転して                   │
│  ├── リクエスト4: リサイズして               │
│  :    （10個ずつ同時に送る）                  │
│  └── リクエスト100: ...                      │
│                                             │
│  → 100人のユーザーが同時アクセスした状況を再現 │
└─────────────────────────────────────────────┘
```

スクリプトのパラメータ：
```bash
./scripts/load-test-image.sh [URL] [並列数] [総リクエスト数]

# 例: 10個ずつ同時に送信、合計100リクエスト
./scripts/load-test-image.sh http://localhost:8080 10 100
```

> **処理結果の画像は保存されない**
>
> 負荷テストの目的は「サーバーに負荷をかける」こと。
> 処理後の画像は自動で破棄され、成功/失敗のステータスだけ表示される。
> ```
> [1] /api/resize?width=200&height=150 → 200 OK
> [2] /api/grayscale → 200 OK
> [3] /api/rotate?angle=90 → 200 OK
> ```

### 準備: metrics-serverをインストール

HPAがCPU使用率を監視するために必要。

```bash
# minikubeの場合
minikube addons enable metrics-server

# 確認（1-2分待つ）
kubectl top pods -n image-api
```

### HPAを適用

```bash
kubectl apply -f k8s/hpa.yaml

# 確認
kubectl get hpa -n image-api
```

これで「CPU使用率50%を超えたら自動でPodを増やす」設定になった。

### 画像処理で負荷をかけてスケールアウトを観察

3つのターミナルを開いて実行する。

#### ターミナル1: HPAとPodを監視
```bash
watch -n 1 'kubectl get hpa,pods -n image-api'
```
（1秒ごとに状態を更新表示）

#### ターミナル2: ポートフォワード
```bash
kubectl port-forward service/image-api 8080:80 -n image-api
```
（Kubernetes内のサービスにlocalhost:8080でアクセスできるようにする）

#### ターミナル3: 画像処理の負荷テスト
```bash
# 大量の画像処理リクエストを送る
./scripts/load-test-image.sh http://localhost:8080 10 100
```

何が起きるか：
```
1. スクリプトが sample.jpg を使って100回リクエストを送る
2. サーバーが画像処理でCPUを使う
3. CPU使用率が50%を超える
4. HPAが検知して「Podを増やせ」と指示
5. 新しいPodが起動する
6. 負荷が分散される
```

### 観察結果

ターミナル1で、CPU使用率が上がり、Podが自動で増えるのが見える：
```
NAME                          REFERENCE              TARGETS     MINPODS   MAXPODS   REPLICAS
hpa/image-api-hpa             Deployment/image-api   78%/50%     1         5         3

NAME                             READY   STATUS
pod/image-api-xxxxx-aaaaa        1/1     Running
pod/image-api-xxxxx-bbbbb        1/1     Running   ← 自動で増えた
pod/image-api-xxxxx-ccccc        1/1     Running   ← 自動で増えた
```

**画像処理のリクエストが増えると、自動でサーバーが増える。**
負荷が下がると、自動でPodが減る。

---

## Step 8: 【体験】ローリングアップデート

サービスを止めずに新バージョンをデプロイ。

例：画像処理の品質を上げたい → コードを変更してデプロイ

### ターミナル1: Podを監視
```bash
kubectl get pods -n image-api -w
```

### ターミナル2: 新バージョンをデプロイ
```bash
# イメージを再ビルド（v2タグ）
docker build -t image-api:v2 .

# デプロイメントのイメージを更新
kubectl set image deployment/image-api image-api=image-api:v2 -n image-api
```

### 観察結果

古いPodが徐々に終了し、新しいPodが起動する：
```
image-api-old-xxxxx   1/1     Running
image-api-old-yyyyy   1/1     Running
image-api-new-aaaaa   0/1     Pending      ← 新バージョン起動開始
image-api-new-aaaaa   1/1     Running      ← 新バージョン準備完了
image-api-old-xxxxx   1/1     Terminating  ← 古いバージョンを終了
...
```

**ユーザーからのリクエストを止めずに、新バージョンに切り替わった。**

---

## クリーンアップ

```bash
# Kubernetesリソース削除
kubectl delete namespace image-api

# minikube停止
minikube stop

# 完全削除したい場合
minikube delete
```

---

## 画像処理API一覧

| エンドポイント | 説明 | パラメータ |
|--------------|------|-----------|
| `GET /health` | ヘルスチェック | - |
| `POST /api/resize` | リサイズ | `width`, `height`, `format` |
| `POST /api/grayscale` | 白黒変換 | `format` |
| `POST /api/rotate` | 回転 | `angle` (90/180/270), `format` |

### 使用例

```bash
# リサイズ（幅200px、高さは自動）
curl -F "image=@photo.jpg" "localhost:8080/api/resize?width=200" -o resized.jpg

# リサイズ（PNG出力）
curl -F "image=@photo.jpg" "localhost:8080/api/resize?width=200&format=png" -o resized.png

# グレースケール
curl -F "image=@photo.jpg" "localhost:8080/api/grayscale" -o gray.jpg

# 90度回転
curl -F "image=@photo.jpg" "localhost:8080/api/rotate?angle=90" -o rotated.jpg
```

---

## Makeコマンド一覧

```bash
make run              # サーバー起動
make build            # バイナリビルド
make generate-image   # テスト画像生成
make load-test        # 負荷テスト

make docker-build     # Dockerイメージビルド
make docker-run       # Dockerコンテナ起動

make k8s-apply        # Kubernetesにデプロイ
make k8s-delete       # Kubernetesから削除
make k8s-status       # 状態確認
make k8s-logs         # ログ確認
```

---

## まとめ

| 課題 | Kubernetesの解決策 | 体験したStep |
|------|-------------------|-------------|
| サーバーが落ちた | 自動復旧 | Step 5 |
| 画像処理リクエスト急増 | 自動スケーリング | Step 7 |
| 新機能をデプロイしたい | ローリングアップデート | Step 8 |

これがKubernetesを使う理由。
画像処理のような重い処理を安定して提供するために、Kubernetesが必要。
