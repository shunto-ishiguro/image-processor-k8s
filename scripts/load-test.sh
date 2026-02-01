#!/bin/bash

# 負荷テストスクリプト
# 使い方: ./scripts/load-test.sh [URL] [並列数] [リクエスト数]

URL=${1:-"http://localhost:8080/health"}
CONCURRENT=${2:-10}
REQUESTS=${3:-100}

echo "=== 負荷テスト開始 ==="
echo "URL: $URL"
echo "並列数: $CONCURRENT"
echo "総リクエスト数: $REQUESTS"
echo ""

# abコマンドがあれば使う、なければcurlでループ
if command -v ab &> /dev/null; then
    ab -n $REQUESTS -c $CONCURRENT "$URL"
else
    echo "Apache Bench (ab) が見つからないため、curlで代用します"
    echo ""

    for i in $(seq 1 $REQUESTS); do
        curl -s -o /dev/null -w "Request $i: %{http_code} (%{time_total}s)\n" "$URL" &

        # 並列数を制御
        if (( i % CONCURRENT == 0 )); then
            wait
        fi
    done
    wait
fi

echo ""
echo "=== 負荷テスト完了 ==="
