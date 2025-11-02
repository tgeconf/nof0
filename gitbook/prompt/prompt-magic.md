# 提示词魔法技巧解析

## 1. 强制结构化输出的魔法

**为什么使用 JSON 格式?**

- **可解析性**: 程序可以自动验证和执行
- **强制完整性**: 缺少字段会导致解析失败,迫使模型完整思考
- **减少幻觉**: 结构化输出比自由文本更可靠

**提示词技巧:**

```markdown
Return your decision as a **valid JSON object** with these exact fields:
```

关键词 "valid" 和 "exact fields" 强化了格式要求。

## 2. 风险管理的元认知设计

**confidence 字段的心理学:**

- 迫使模型进行"元认知"(thinking about thinking)
- 低 confidence → 自动降低仓位
- 创造"自我怀疑"机制,防止过度自信

**invalidation_condition 的作用:**

- 预先承诺退出条件,避免"移动止损"
- 强制模型思考"什么情况下我错了?"
- 类似于人类交易者的"交易日志"

## 3. 数据顺序的反复强调

**为什么多次重复 "OLDEST → NEWEST"?**

LLM 在处理时间序列时有天然的混淆倾向:

- 训练数据中时间顺序不一致
- 注意力机制对位置不敏感
- 容易把"最新"和"最旧"搞反

**解决方案:**

1. 在 System Prompt 中说明一次
2. 在 User Prompt 开头用 ⚠️ 警告
3. 在每个数据块前再次提醒
4. 使用视觉标记(大写、粗体、表情符号)

## 4. 多时间框架的认知负载管理

**3分钟 + 4小时的双重视角:**

- **3分钟**: 短期入场时机,噪音较多
- **4小时**: 中期趋势背景,信号更可靠

**提示词设计:**

```markdown
**Intraday series (3-minute intervals):** [短期数据]
**Longer-term context (4-hour timeframe):** [中期数据]
```

明确标注时间框架,避免混淆。

## 5. 费用意识的植入

**为什么强调交易费用?**

- LLM 默认倾向于"过度交易"(更多动作 = 更积极?)
- 明确提及费用可以抑制无意义的频繁交易

**提示词技巧:**

```markdown
Trading Fees: ~0.02-0.05% per trade
⚠️ Avoid over-trading; fees will erode profits on small, frequent trades
```

## 6. 无状态设计的哲学

**每次调用独立,无历史记忆:**

- 测试模型的即时决策能力
- 避免"路径依赖"和"沉没成本谬误"
- ⚠️ 无法学习和改进(除非通过 Sharpe Ratio 反馈)

这是 Season 1 的限制,未来可能引入:
- 短期记忆(最近 N 次交易)
- 长期学习(跨 session 的策略优化)
