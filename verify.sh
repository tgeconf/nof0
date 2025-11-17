#!/bin/bash
# 验证前端后端连接

echo "🔍 验证部署状态..."
echo ""

# 1. 检查后端
echo "1️⃣ 检查后端服务..."
if curl -s http://localhost:8888/api/leaderboard > /dev/null 2>&1; then
    echo "   ✅ 后端服务正常 (http://localhost:8888)"
else
    echo "   ❌ 后端服务未运行或无法访问"
    exit 1
fi

# 2. 检查前端
echo "2️⃣ 检查前端服务..."
if curl -s http://localhost:3000 > /dev/null 2>&1; then
    echo "   ✅ 前端服务正常 (http://localhost:3000)"
else
    echo "   ❌ 前端服务未运行或无法访问"
    exit 1
fi

# 3. 检查 API 代理
echo "3️⃣ 检查 API 代理..."
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/api/nof1/leaderboard 2>/dev/null)
if [ "$RESPONSE" = "200" ]; then
    echo "   ✅ API 代理正常 (返回 200)"
else
    echo "   ❌ API 代理异常 (返回 $RESPONSE)"
    exit 1
fi

# 4. 检查数据
echo "4️⃣ 检查数据返回..."
DATA=$(curl -s http://localhost:3000/api/nof1/leaderboard 2>/dev/null)
if echo "$DATA" | grep -q "leaderboard"; then
    echo "   ✅ 数据正常返回"
else
    echo "   ❌ 数据格式异常"
    exit 1
fi

echo ""
echo "🎉 所有检查通过！前端和后端已成功连接！"
