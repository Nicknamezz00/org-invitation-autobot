#!/bin/bash

PORT=$1

if [ -z "$PORT" ]; then
  echo "❌ 用法: ./free-port-ss.sh <端口号>"
  exit 1
fi

# 查询使用该端口的 PID
INFO=$(ss -ltnp 2>/dev/null | grep ":$PORT ")

if [ -z "$INFO" ]; then
  echo "✅ 端口 $PORT 没有被占用"
  exit 0
fi

echo "⚠️ 端口 $PORT 被以下进程占用："
echo "$INFO"

# 从 ss 输出中提取 PID
PID=$(echo "$INFO" | grep -oP 'pid=\K[0-9]+')

if [ -z "$PID" ]; then
  echo "❌ 无法提取进程 PID，可能没有权限（尝试用 sudo）"
  exit 1
fi

# 二次确认
read -p "是否杀掉该进程？[y/N]: " CONFIRM
if [[ "$CONFIRM" == "y" || "$CONFIRM" == "Y" ]]; then
  kill -9 "$PID"
  echo "✅ 已杀掉进程 $PID，端口 $PORT 已释放"
else
  echo "❎ 未执行操作"
fi

