# Strategy Simulation Guide

## Strategy Types

### 1. Swing Trading (摆动交易)

**Profile**:
- **Time Horizon**: 2-10 days
- **Frequency**: 3-5 trades per week
- **Focus**: Capture medium-term price swings
- **Suitable for**: Part-time traders,上班族

**Advantages**:
- ✅ Lower transaction costs
- ✅ Less time-intensive
- ✅ Captures meaningful moves
- ✅ Good risk/reward ratio

**Disadvantages**:
- ❌ Overnight risk
- ❌ Requires patience
- ❌ Misses long-term trends

**Key Indicators**:
- 20/60-day moving averages
- RSI (14-period)
- MACD daily
- Volume trends

**Entry Rules**:
- Stock in uptrend (20MA > 60MA)
- Pullback to support
- RSI 30-40 (oversold in uptrend)
- Volume drying up
- MACD turning up

**Exit Rules**:
- RSI 70+ (overbought)
- Break below 20MA
- MACD bearish crossover
- Target: 1.5-2x risk

**Position Sizing**:
- Max 10% per position
- 2% risk per trade
- 3-5 positions max

---

### 2. Day Trading (日内交易)

**Profile**:
- **Time Horizon**: Intraday (minutes to hours)
- **Frequency**: 5-20+ trades per day
- **Focus**: Capture intraday volatility
- **Suitable for**: Full-time traders

**Advantages**:
- ✅ No overnight risk
- ✅ High frequency opportunities
- ✅ Quick feedback loop
- ✅ Capital efficiency

**Disadvantages**:
- ❌ High stress
- ❌ Time intensive
- ❌ High transaction costs
- ❌ Steep learning curve

**Key Indicators**:
- 5/15/60-minute charts
- VWAP (Volume Weighted Average Price)
- Level 2 data
- Volume spikes

**Entry Rules**:
- Clear intraday trend
- Volume confirmation
- Break of key levels
- Momentum confirmation

**Exit Rules**:
- Fixed profit target (1-3%)
- Time-based (end of day)
- Stop loss (0.5-1%)
- Momentum exhaustion

**Position Sizing**:
- Max 5% per position
- 0.5-1% risk per trade
- Many small trades

---

### 3. Position Trading (持仓交易)

**Profile**:
- **Time Horizon**: Weeks to months
- **Frequency**: 1-3 trades per month
- **Focus**: Capture major trends
- **Suitable for**: Patient investors

**Advantages**:
- ✅ Captures major moves
- ✅ Low transaction costs
- ✅ Less time required
- ✅ Works well with fundamentals

**Disadvantages**:
- ❌ Large drawdowns possible
- ❌ Requires big patience
- ❌ Many small losses
- ❌ Opportunity cost

**Key Indicators**:
- Weekly/monthly charts
- 50/200-day moving averages
- Trend lines
- Fundamental catalysts

**Entry Rules**:
- Strong primary trend
- Fundamental improvement
- Break of major resistance
- Sector rotation support

**Exit Rules**:
- Trend reversal signals
- Fundamental deterioration
- Major resistance reached
- Time-based review (quarterly)

**Position Sizing**:
- Max 20% per position
- 5% risk per trade
- 3-5 positions max

---

### 4. Trend Trading (波段交易)

**Profile**:
- **Time Horizon**: Days to weeks
- **Frequency**: 5-10 trades per month
- **Focus**: Ride established trends
- **Suitable for**: Most traders

**Advantages**:
- ✅ Clear entry/exit rules
- ✅ Good risk/reward
- ✅ Works in trending markets
- ✅ Flexible time frame

**Disadvantages**:
- ❌ Whipsaws in sideways markets
- ❌ Misses early trend
- ❌ Requires trend confirmation

**Key Indicators**:
- 20/50/200-day MAs
- ADX (trend strength)
- Channel lines
- Breakout/breakdown levels

**Entry Rules**:
- Trend confirmed (ADX > 25)
- Pullback to support
- Volume increasing
- Multiple time frame alignment

**Exit Rules**:
- Trend weakening (ADX falling)
- Break of trend line
- Reversal patterns
- Risk/reward target reached

**Position Sizing**:
- Max 15% per position
- 3% risk per trade
- 4-6 positions max

---

## Risk Levels

### Conservative
```
Profile:
- Risk tolerance: Low
- Capital preservation: Priority
- Return expectation: 8-12% annually

Strategy Mix:
- 70% Position Trading
- 20% Swing Trading
- 10% Cash

Parameters:
- Single position: 5-10%
- Stop loss: 5-8%
- Max drawdown: 10%
- Leverage: None
```

### Balanced
```
Profile:
- Risk tolerance: Medium
- Capital growth: Priority
- Return expectation: 12-20% annually

Strategy Mix:
- 40% Position Trading
- 40% Swing Trading
- 20% Trend Trading

Parameters:
- Single position: 10-15%
- Stop loss: 8-10%
- Max drawdown: 15%
- Leverage: Minimal (1:1)
```

### Aggressive
```
Profile:
- Risk tolerance: High
- Capital growth: Priority
- Return expectation: 20%+ annually

Strategy Mix:
- 30% Swing Trading
- 30% Trend Trading
- 40% Day Trading

Parameters:
- Single position: 15-20%
- Stop loss: 10-15%
- Max drawdown: 25%
- Leverage: Moderate (1:2)
```

## Market Condition Matching

### Bull Market
```
Best Strategies:
- Position Trading (ride the trend)
- Swing Trading (buy dips)
- Trend Trading (momentum)

Avoid:
- Short selling
- Excessive hedging
```

### Bear Market
```
Best Strategies:
- Short Swing Trading
- Cash preservation
- Defensive positioning

Avoid:
- Long position trading
- Buying dips aggressively
```

### Sideways Market
```
Best Strategies:
- Range trading (buy low, sell high)
- Day trading (volatility plays)
- Avoid trend strategies

Avoid:
- Trend following
- Breakout strategies
```

## Performance Metrics

### Key Metrics to Track

#### Win Rate
```
Formula: Winning Trades / Total Trades × 100%

Target by Strategy:
- Day Trading: 50-60%
- Swing Trading: 40-50%
- Position Trading: 30-40%

Note: Lower win rate can be OK with high reward/risk
```

#### Risk/Reward Ratio
```
Formula: Average Win / Average Loss

Target: > 2:1
Minimum: 1.5:1

Example:
- Average win: 8%
- Average loss: 3%
- Ratio: 2.67:1 ✓
```

#### Profit Factor
```
Formula: Gross Profit / Gross Loss

Target: > 1.5
Excellent: > 2.0
```

#### Maximum Drawdown
```
Formula: Peak to trough decline

Target: < 15%
Acceptable: < 20%
Danger: > 25%
```

#### Expectancy
```
Formula: (Win% × Avg Win) - (Loss% × Avg Loss)

Example:
- Win rate: 45%
- Avg win: 8%
- Avg loss: 3%

  = =  .>

.........
  . to....</ .
><...
  ...    .. ``   :.


 .

   .
`` +













      .





:


.



























.
















 /









  (















.

 /
.  **  ( 0 / rend  3 1 1 . 3  10%
- 10% cash

```

### Performance Adjustment
```
Profile: 30% Position 30% Trend 30% + 10% cash

Adjust based on:
- Market conditions
- Personal risk tolerance
- Strategy performance
- Capital size
```

## Strategy Evaluation Framework

### Before Testing
```
1. Define clear rules
2. Set performance benchmarks
3. Determine sample size (min 30 trades)
4. Choose backtesting period
5. Account for all costs
```

### During Testing
```
1. Track all trades meticulously
2. Follow rules strictly
3. Note market conditions
4. Record emotions/decisions
5. Monitor drawdowns
```

### After Testing
```
1. Calculate all metrics
2. Compare to benchmarks
3. Identify strengths/weaknesses
4. Optimize parameters (carefully)
5. Plan live implementation
```

## Common Pitfalls

### Strategy Hopping
```
Problem: Switching strategies after few losses
Solution: Commit to 30+ trades before evaluation
```

### Over-Optimization
```
Problem: Tweaking for perfect historical fit
Solution: Keep rules simple, test out-of-sample
```

### Ignoring Market Context
```
Problem: Same strategy in all markets
Solution: Match strategy to market conditions
```

### Poor Risk Management
```
Problem: Large losses wipe out gains
Solution: Strict position sizing and stops
```

### Emotional Trading
```
Problem: Deviating from plan
Solution: Written rules, automated execution
```

## Implementation Checklist

### Before Trading
- [ ] Strategy defined in writing
- [ ] Rules clear and specific
- [ ] Risk parameters set
- [ ] Capital allocated
- [ ] Trading journal ready
- [ ] Performance benchmarks set

### During Trading
- [ ] Follow rules strictly
- [ ] Record every trade
- [ ] Track emotions
- [ ] Monitor drawdown
- [ ] Review weekly

### After Trading
- [ ] Calculate metrics
- [ ] Compare to benchmarks
- [ ] Identify improvements
- [ ] Adjust if needed
- [ ] Continue or switch

---

**Remember**: No strategy works all the time. The key is finding a strategy that fits your personality, risk tolerance, and lifestyle, then executing it with discipline.
