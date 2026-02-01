#!/bin/bash

# Podの状態をリアルタイムで監視
# 別ターミナルで実行しながら負荷テストすると、Podが増減するのが見える

echo "=== Pod監視中 (Ctrl+Cで終了) ==="
echo ""

kubectl get pods -n image-api -w
