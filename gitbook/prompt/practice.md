# 实战应用建议

## 如果你想复现或改进 nof1.ai

**可以调整的参数:**

1. **决策频率**: 2-3分钟 → 5-10分钟(降低交易频率)
2. **杠杆限制**: 1-20x → 1-5x(降低风险)
3. **资产范围**: 6个币 → 扩展到更多或更少
4. **技术指标**: 增加布林带、成交量分布等
5. **风险管理**: 强制最大回撤限制、单日亏损熔断

**提示词改进方向:**

```markdown
# 增加回撤控制
- **Maximum Drawdown Limit**: If account value drops >15% from peak, STOP trading
- **Daily Loss Limit**: If daily loss exceeds 5%, switch to "hold" only mode

# 增加相关性分析
- **Correlation Check**: Before entering new position, check correlation with existing positions
- **Diversification Rule**: No more than 2 positions in highly correlated assets (>0.7)

# 增加市场状态识别
- **Market Regime Detection**:
  - Trending (use trend-following strategies)
  - Ranging (use mean-reversion strategies)
  - High Volatility (reduce position sizes)
```

## 提示词测试清单

在部署之前,测试以下场景:

- [ ] **边界条件**: 账户余额为0时的行为
- [ ] **极端市场**: 价格暴涨/暴跌时的反应
- [ ] **数据异常**: 缺失数据或异常值的处理
- [ ] **JSON 格式**: 输出是否总是有效的 JSON
- [ ] **风险计算**: 仓位大小和止损是否合理
- [ ] **时间序列**: 是否正确理解数据顺序
