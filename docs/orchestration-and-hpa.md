# オーケストレーションと自動スケーリング (HPA)

Kubernetesの「オーケストレーション（状態維持）」と「HPA（自動スケーリング）」は別の仕組み。
この2つの違いと関係を整理する。

---

## 目次

1. [オーケストレーション = 宣言した状態を維持する](#オーケストレーション--宣言した状態を維持する)
2. [HPA = メトリクスに基づいて自動で増減する](#hpa--メトリクスに基づいて自動で増減する)
3. [metrics-serverとは](#metrics-serverとは)
4. [なぜメトリクスはKubernetes本体に含まれないのか](#なぜメトリクスはkubernetes本体に含まれないのか)
5. [全体の関係図](#全体の関係図)
6. [hpa.yamlの設定内容](#hpayamlの設定内容)
7. [スケーリングの流れ](#スケーリングの流れ)
8. [確認コマンド](#確認コマンド)

---

## オーケストレーション = 宣言した状態を維持する

Kubernetesのコア機能は**「宣言した状態を維持すること」**。

```yaml
# deployment.yaml
spec:
  replicas: 3  # ← 「Podを3つ動かしてね」という宣言
```

この場合、Kubernetesは**常にPodを3つ維持する**。メトリクスは不要。

### オーケストレーションがやること

| 状況 | Kubernetesの動作 |
|------|----------------|
| Podが1つ落ちた | 自動で1つ作って3に戻す（自己修復） |
| 負荷が高い | **何もしない**（3のまま） |
| 負荷がゼロ | **何もしない**（3のまま） |

```mermaid
graph LR
    R["replicas: 3"] --> P1["Pod 1 ✅ 動いてる"]
    R --> P2["Pod 2 ✅ 動いてる"]
    R --> P3["Pod 3 ❌ 落ちた！"]
    P3 -->|自動で新しいPodを作る| P4["Pod 3' ✅ 復旧"]
```

### 手動スケーリング

Pod数を変えたければ手動でコマンドを打つ：

```bash
# 5台に増やす
kubectl scale deployment image-api --replicas=5 -n image-api

# 1台に減らす
kubectl scale deployment image-api --replicas=1 -n image-api
```

これがオーケストレーションの範囲。**「何台にするか」は人間が決める。**

---

## HPA = メトリクスに基づいて自動で増減する

HPA（Horizontal Pod Autoscaler）は、CPU使用率などのメトリクスを見て**「何台にするか」を自動で判断する仕組み**。

```mermaid
graph LR
    O["オーケストレーション\n「3台を維持しろ」"] --> OR["ずっと3台"]
    H["HPA\n「CPU 50%を目標に自動調整しろ」"] --> HR["負荷に応じて1〜5台"]
```

### オーケストレーションとHPAの違い

| | オーケストレーション | HPA |
|--|-------------------|-----|
| Pod数の決め方 | 人間が `replicas` に書く | メトリクスから自動計算 |
| メトリクスが必要？ | 不要 | **必要** |
| 負荷が増えたら | 何もしない | Podを増やす |
| 負荷が減ったら | 何もしない | Podを減らす |
| Kubernetes本体の機能？ | はい | はい（ただしメトリクス収集は別） |

HPAは「replicas の値を自動で書き換えてくれるコントローラー」と思えばいい。
書き換えた後の「その台数を維持する」部分はオーケストレーションが担当する。

---

## metrics-serverとは

metrics-serverは、kubeletからCPU/メモリの使用量を収集し、Metrics APIとして公開するコンポーネント。

- Kubernetes公式プロジェクトが提供する**アドオン**（本体とは別にインストールする拡張機能）
- Kubernetes本体には**含まれていない**（別途インストールが必要）
- HPAや `kubectl top` コマンドがこのAPIを使う

### クラスター全体で1つ

metrics-serverはPodごとに1つ付くわけではなく、**クラスター全体に対して1つ**だけ動く。

各Nodeにいる**kubelet**（Podを管理するエージェント）がリソース情報を持っていて、
metrics-serverがそれをまとめて集める仕組み。

```mermaid
flowchart TD
    subgraph NodeA["Node A"]
        KA["kubelet"] --> PA1["Pod 1の情報"]
        KA --> PA2["Pod 2の情報"]
    end
    subgraph NodeB["Node B"]
        KB["kubelet"] --> PB1["Pod 3の情報"]
        KB --> PB2["Pod 4の情報"]
    end
    KA --> MS["metrics-server（1つ）\n「全Podの情報まとめたよ」"]
    KB --> MS
    MS --> HPA["HPA\n「じゃあPod増やすか」"]
```

このプロジェクトの場合：

```mermaid
graph TD
    subgraph K8s["Kubernetesクラスター"]
        MS["metrics-server\n（kube-system / クラスターに1つ）"]
        MS -->|CPU/メモリ収集| P1["image-api Pod 1"]
        MS -->|CPU/メモリ収集| P2["image-api Pod 2"]
        MS -->|CPU/メモリ収集| P3["image-api Pod 3"]
    end
```

```bash
# metrics-serverのPodを確認
kubectl get pods -n kube-system | grep metrics
```

### インストール方法

```bash
# minikubeの場合
minikube addons enable metrics-server

# それ以外の環境
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# 確認（1-2分待つ）
kubectl top pods -n image-api
```

---

## なぜメトリクスはKubernetes本体に含まれないのか

直感的には「スケーリングするならメトリクスは必須では？」と思う。
しかしKubernetesの設計思想は**「コア機能は最小限にして、残りはプラグインで」**。

### 理由1: オーケストレーションにメトリクスは不要

Kubernetes本体が担うのは「宣言された状態を維持する」こと。

- `replicas: 3` と書いてあれば3台維持する
- Podが落ちたら再作成する

これにメトリクスは必要ない。メトリクスが必要になるのはHPA（自動スケーリング）を使うときだけ。

### 理由2: メトリクスの収集方法は環境によって違う

```mermaid
graph LR
    Cloud["クラウド環境"] --> DD["Datadog / CloudWatch"]
    Large["大規模環境"] --> Prom["Prometheus"]
    Learn["学習環境"] --> MS["metrics-server で十分"]
```

metrics-serverを本体に組み込むと「使わないのに動いてる」「別のに差し替えたい」ときに困る。

### 理由3: インターフェースと実装の分離

Kubernetesは **Metrics API（インターフェース）だけ定義** して、実装は差し替え可能にしている。

```mermaid
graph LR
    MS["metrics-server"] --> API["Metrics API"]
    Prom["Prometheus"] --> API
    DD["Datadog"] --> API
    API --> HPA["HPA（判断）"]
    HPA --> Deploy["Deployment（実行）"]
```

Linuxカーネルがファイルシステムの仕組みは提供するけど、ext4/xfs/btrfsは別モジュール、というのに近い。

---

## 全体の関係図

```mermaid
graph LR
    subgraph Addon["アドオン"]
        MS["metrics-server\n（メトリクス収集）\nkubeletからCPU/メモリ収集"]
    end
    subgraph Core["Kubernetes本体"]
        HPA["HPA（判断）\nCPU 80%だ！\n→ 5台に増やそう"]
        Deploy["Deployment（状態維持）\nreplicas: 3\n→ 3台を維持"]
    end
    MS -->|数値を提供| HPA
    HPA -->|書き換え| Deploy
```

metrics-serverがないと:
- HPAは判断材料がない → 自動スケーリングできない
- Deploymentは問題なく動く → replicas固定で維持

---

## hpa.yamlの設定内容

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

### 設定値の意味

| 設定 | 意味 |
|------|------|
| `scaleTargetRef` | どのDeploymentを対象にするか |
| `minReplicas: 1` | 最低1台は動かす |
| `maxReplicas: 5` | 最大5台まで増やす |
| `averageUtilization: 50` | 平均CPU使用率を50%に保つよう調整 |

### CPU使用率とPod数の関係

```mermaid
graph LR
    C20["CPU 20%"] --> R1["Pod 1台で十分"]
    C60["CPU 60%"] --> R2["50%に近づけるため\nPod増やす"]
    C80["CPU 80%"] --> R3["もっとPod増やす"]
    C30["CPU 30%"] --> R4["Pod減らす\n（でも最低1台）"]
```

---

## スケーリングの流れ

```mermaid
flowchart TD
    A["負荷テスト開始"] --> B["CPU使用率上昇（80%）"]
    B --> C["HPAが検知"]
    C --> D["Podを増やす（1台→3台）"]
    D --> E["負荷分散されて\nCPU使用率低下（50%前後）"]
    E --> F["負荷テスト終了"]
    F --> G["CPU使用率低下（10%）"]
    G --> H["Podを減らす（3台→1台）"]
```

---

## 確認コマンド

```bash
# HPA の状態を確認
kubectl get hpa -n image-api

# リアルタイム監視
watch kubectl get hpa,pods -n image-api

# Pod のリソース使用量を確認（metrics-server必要）
kubectl top pods -n image-api
```
