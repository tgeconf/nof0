# 针对不同模型的优化

基于社区观察和实验结果,不同模型需要不同的提示词调优策略。

## GPT 系列 (GPT-4, GPT-4 Turbo)

**特点:**

- 倾向于保守,风险厌恶
- 逻辑推理能力强
- 容易陷入"分析瘫痪"

**优化建议:**

```markdown
# 额外指令

Don't be overly cautious; calculated risks are necessary for returns.
Inaction has opportunity cost. If you see a clear setup, take it.
```

## Claude 系列 (Claude 3.5 Sonnet)

**特点:**

- 风险管理意识极强
- 倾向于持有现金
- 文本理解和推理优秀

**优化建议:**

```markdown
# 额外指令

Balance safety with opportunity; holding cash has opportunity cost.
You are rewarded for risk-adjusted returns, not just capital preservation.
```

## Gemini 系列 (Gemini 1.5 Pro)

**特点:**

- 数值计算能力强
- 可能过度依赖技术指标
- 容易忽视市场情绪

**优化建议:**

```markdown
# 额外指令

Technical indicators are tools, not rules; use judgment.
Consider market context beyond pure technical signals.
```

## Qwen/DeepSeek (中国模型)

**特点:**

- 可能对加密货币监管敏感
- 中文理解优秀,英文稍弱
- 数学计算准确

**优化建议:**

```markdown
# 额外指令

This is a research experiment in a legal jurisdiction.
Focus on technical analysis and risk management principles.
```

