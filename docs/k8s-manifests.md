# Kubernetes マニフェスト解説

`k8s/` フォルダにあるYAMLファイルの詳細な説明。

---

## 目次

1. [YAMLファイルとは](#yamlファイルとは)
2. [適用する順番](#適用する順番)
3. [namespace.yaml](#namespaceyaml)
4. [configmap.yaml](#configmapyaml)
5. [secret.yaml](#secretyaml)
6. [deployment.yaml](#deploymentyaml)
7. [service.yaml](#serviceyaml)
8. [hpa.yaml](#hpayaml)

---

## YAMLファイルとは

Kubernetesへの「指示書」。

```
「こういうアプリを動かしてね」
「サーバー2台で動かしてね」
「ポート8080で公開してね」
```

これをYAML形式で書いたもの。

---

## 適用する順番

依存関係があるので、順番が大事：

```
1. namespace.yaml   ← まず「部屋」を作る
2. configmap.yaml   ← 設定を用意
3. secret.yaml      ← 機密情報を用意
4. deployment.yaml  ← アプリを起動（↑の設定を使う）
5. service.yaml     ← 外部に公開
6. hpa.yaml         ← 自動スケーリング設定
```

コマンド：
```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/hpa.yaml
```

または `make k8s-apply` で一括適用。

---

## namespace.yaml

### 役割

「部屋」を作る。リソースをグループ分けするため。

### ファイル内容

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: image-api        # 部屋の名前
  labels:
    app: image-processing
```

### 行ごとの説明

| 行 | 意味 |
|----|------|
| `apiVersion: v1` | 使うAPIのバージョン |
| `kind: Namespace` | 「これはNamespaceの定義です」 |
| `metadata:` | このリソースの情報 |
| `name: image-api` | Namespaceの名前 |
| `labels:` | タグ付け（検索しやすくするため） |

### なぜ必要？

```
Kubernetesクラスター
├── default（デフォルトの部屋）
├── kube-system（システム用）
└── image-api（このプロジェクト用）← これを作る
```

他のプロジェクトと混ざらないように分ける。

### 確認コマンド

```bash
kubectl get namespaces
```

---

## configmap.yaml

### 役割

設定値を外部ファイルで管理。コードを変えずに設定だけ変更できる。

### ファイル内容

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: image-api-config
  namespace: image-api      # どの部屋に作るか
data:
  PORT: "8080"              # サーバーのポート
  LOG_LEVEL: "info"         # ログレベル
  MAX_IMAGE_SIZE: "10485760"  # 最大画像サイズ（10MB）
```

### 行ごとの説明

| 行 | 意味 |
|----|------|
| `kind: ConfigMap` | 「これは設定ファイルです」 |
| `namespace: image-api` | image-api部屋に作る |
| `data:` | 設定値の一覧 |
| `PORT: "8080"` | 環境変数 PORT=8080 |

### どう使われる？

Deploymentで参照される：

```yaml
envFrom:
  - configMapRef:
      name: image-api-config  # ← これを読み込む
```

アプリからは環境変数として見える：

```go
port := os.Getenv("PORT")  // → "8080"
```

### なぜ必要？

設定をコードから分離できる：

```
本番環境: PORT=80, LOG_LEVEL=error
開発環境: PORT=8080, LOG_LEVEL=debug
```

コードは同じ、設定だけ変える。

### 確認コマンド

```bash
kubectl get configmap -n image-api
kubectl describe configmap image-api-config -n image-api
```

---

## secret.yaml

### 役割

パスワードやAPIキーなど、機密情報を管理。

### ファイル内容

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: image-api-secret
  namespace: image-api
type: Opaque
data:
  API_KEY: eW91ci1zZWNyZXQta2V5   # base64エンコード済み
```

### ConfigMapとの違い

| | ConfigMap | Secret |
|--|-----------|--------|
| 用途 | 普通の設定 | 機密情報 |
| 値 | そのまま | base64エンコード |
| 例 | PORT, LOG_LEVEL | パスワード, APIキー |

### base64エンコードとは

```bash
# エンコード（保存するとき）
echo -n "your-secret-key" | base64
# → eW91ci1zZWNyZXQta2V5

# デコード（確認するとき）
echo "eW91ci1zZWNyZXQta2V5" | base64 -d
# → your-secret-key
```

**注意**: base64は暗号化ではない。簡単に戻せる。本番では別の方法（Vault等）を使う。

### 確認コマンド

```bash
kubectl get secret -n image-api

# 値を確認（デコード）
kubectl get secret image-api-secret -n image-api -o jsonpath='{.data.API_KEY}' | base64 -d
```

---

## deployment.yaml

### 役割

アプリ（Pod）をどう動かすか定義。一番重要なファイル。

### ファイル内容（コメント付き）

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: image-api
  namespace: image-api
  labels:
    app: image-api
spec:
  replicas: 2                    # Podを2つ動かす
  selector:
    matchLabels:
      app: image-api             # このラベルのPodを管理
  template:                      # Podのテンプレート
    metadata:
      labels:
        app: image-api
    spec:
      containers:
        - name: image-api
          image: image-api:latest       # Dockerイメージ
          imagePullPolicy: IfNotPresent # ローカルにあれば使う
          ports:
            - containerPort: 8080       # コンテナのポート
          envFrom:
            - configMapRef:
                name: image-api-config  # ConfigMapを環境変数に
            - secretRef:
                name: image-api-secret  # Secretを環境変数に
          resources:
            requests:                   # 最低限必要なリソース
              memory: "64Mi"
              cpu: "100m"
            limits:                     # 最大使用量
              memory: "256Mi"
              cpu: "500m"
          livenessProbe:               # 生存確認
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:              # 準備完了確認
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
```

### 重要な部分を詳しく

#### replicas（レプリカ数）

```yaml
replicas: 2
```

同じPodを何個動かすか。2なら2台のサーバーが起動する。

```
replicas: 2
  → Pod 1（画像処理API）
  → Pod 2（画像処理API）
```

#### resources（リソース制限）

```yaml
resources:
  requests:          # 「最低これだけ欲しい」
    memory: "64Mi"   # メモリ64MB
    cpu: "100m"      # CPU 0.1コア（1000m = 1コア）
  limits:            # 「これ以上は使わせない」
    memory: "256Mi"
    cpu: "500m"
```

なぜ必要？
- 1つのPodが暴走しても、他に影響しない
- HPAがスケーリング判断に使う

#### livenessProbe（生存確認）

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5   # 起動後5秒待ってから確認開始
  periodSeconds: 10        # 10秒ごとに確認
```

「アプリが生きてるか」を確認。失敗したらPodを再起動する。

#### readinessProbe（準備完了確認）

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
```

「リクエストを受け付けられるか」を確認。失敗したらトラフィックを送らない。

```
livenessProbe失敗  → Podを再起動
readinessProbe失敗 → そのPodにリクエストを送らない（再起動はしない）
```

### 確認コマンド

```bash
kubectl get deployment -n image-api
kubectl describe deployment image-api -n image-api

# Podの状態
kubectl get pods -n image-api
```

---

## service.yaml

### 役割

Podへのアクセス方法を定義。ロードバランサーの役割。

### なぜ必要？

Podは起動するたびにIPアドレスが変わる：

```
Pod再起動前: 10.0.0.5
Pod再起動後: 10.0.0.8  ← IPが変わった！
```

Serviceは固定のアドレスを提供する：

```
Service: image-api (固定)
  └→ Pod 1 (10.0.0.5)
  └→ Pod 2 (10.0.0.6)
  └→ Pod 3 (10.0.0.7)
```

### ファイル内容

2つのServiceを定義している：

#### ClusterIP（クラスター内部用）

```yaml
apiVersion: v1
kind: Service
metadata:
  name: image-api
  namespace: image-api
spec:
  type: ClusterIP           # クラスター内部からのみアクセス可能
  selector:
    app: image-api          # このラベルのPodに転送
  ports:
    - port: 80              # Serviceのポート
      targetPort: 8080      # Podのポート
```

```
クラスター内部から:
http://image-api.image-api.svc.cluster.local:80
  → 自動でPodに振り分け
```

#### NodePort（外部公開用）

```yaml
apiVersion: v1
kind: Service
metadata:
  name: image-api-nodeport
spec:
  type: NodePort
  selector:
    app: image-api
  ports:
    - port: 80
      targetPort: 8080
      nodePort: 30080       # Nodeの30080ポートで公開
```

```
外部から:
http://<NodeのIP>:30080
  → Serviceに到達
  → Podに振り分け
```

### Serviceの種類

| 種類 | 用途 | アクセス元 |
|------|------|-----------|
| ClusterIP | 内部通信 | クラスター内のみ |
| NodePort | 開発/テスト | 外部からNodeのポートで |
| LoadBalancer | 本番 | クラウドのロードバランサー経由 |

### 確認コマンド

```bash
kubectl get service -n image-api

# ポートフォワード（ローカルからアクセス）
kubectl port-forward service/image-api 8080:80 -n image-api
```

---

## hpa.yaml

### 役割

負荷に応じてPod数を自動調整。HPA = Horizontal Pod Autoscaler。

### ファイル内容

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: image-api-hpa
  namespace: image-api
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: image-api          # このDeploymentを対象に
  minReplicas: 1             # 最小Pod数
  maxReplicas: 5             # 最大Pod数
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50   # CPU使用率50%を目標
```

### 行ごとの説明

| 行 | 意味 |
|----|------|
| `scaleTargetRef` | どのDeploymentを対象にするか |
| `minReplicas: 1` | 最低1台は動かす |
| `maxReplicas: 5` | 最大5台まで増やす |
| `averageUtilization: 50` | 平均CPU使用率を50%に保つ |

### どう動く？

```
CPU使用率 20% → Pod 1台で十分
CPU使用率 60% → 50%に近づけるため Pod増やす
CPU使用率 80% → もっとPod増やす
CPU使用率 30% → Pod減らす（でも最低1台）
```

```
負荷テスト開始
  ↓
CPU使用率上昇（80%）
  ↓
HPAが検知
  ↓
Podを増やす（1台→3台）
  ↓
負荷分散されてCPU使用率低下（50%前後）
  ↓
負荷テスト終了
  ↓
CPU使用率低下（10%）
  ↓
Podを減らす（3台→1台）
```

### 確認コマンド

```bash
kubectl get hpa -n image-api

# リアルタイム監視
watch kubectl get hpa,pods -n image-api
```

---

## まとめ

| ファイル | 役割 | 一言で |
|---------|------|--------|
| namespace.yaml | 部屋を作る | リソースのグループ分け |
| configmap.yaml | 設定を外部化 | PORT, LOG_LEVELなど |
| secret.yaml | 機密情報を管理 | パスワード, APIキー |
| deployment.yaml | アプリを動かす | Pod数、リソース、ヘルスチェック |
| service.yaml | 外部に公開 | ロードバランサー |
| hpa.yaml | 自動スケーリング | 負荷に応じてPod増減 |
