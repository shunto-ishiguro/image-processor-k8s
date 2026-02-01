#!/bin/bash

# 画像処理APIへの負荷テスト
# 使い方: ./scripts/load-test-image.sh [URL] [並列数] [リクエスト数]

BASE_URL=${1:-"http://localhost:8080"}
CONCURRENT=${2:-5}
REQUESTS=${3:-50}
IMAGE_PATH="testdata/sample.jpg"

# 画像があるか確認
if [ ! -f "$IMAGE_PATH" ]; then
    echo "Error: $IMAGE_PATH が見つかりません"
    echo "先に make generate-image を実行してください"
    exit 1
fi

echo "=== 画像処理API 負荷テスト ==="
echo "URL: $BASE_URL"
echo "並列数: $CONCURRENT"
echo "総リクエスト数: $REQUESTS"
echo "テスト画像: $IMAGE_PATH"
echo ""

# 3種類のAPIをランダムに叩く
ENDPOINTS=(
    "/api/resize?width=200&height=150"
    "/api/grayscale"
    "/api/rotate?angle=90"
)

success=0
fail=0

for i in $(seq 1 $REQUESTS); do
    # ランダムにエンドポイントを選択
    endpoint=${ENDPOINTS[$((RANDOM % 3))]}
    url="${BASE_URL}${endpoint}"

    # バックグラウンドでリクエスト
    (
        result=$(curl -s -o /dev/null -w "%{http_code}" -X POST -F "image=@$IMAGE_PATH" "$url")
        if [ "$result" = "200" ]; then
            echo "[$i] $endpoint → 200 OK"
        else
            echo "[$i] $endpoint → $result FAIL"
        fi
    ) &

    # 並列数を制御
    if (( i % CONCURRENT == 0 )); then
        wait
    fi
done

wait

echo ""
echo "=== 負荷テスト完了 ==="
