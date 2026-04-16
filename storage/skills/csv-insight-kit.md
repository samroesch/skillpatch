---
name: csv-insight-kit
description: Analyzes CSV data and produces concise findings, patterns, and chart guidance. Use when the user shares CSV data or asks for data analysis, trends, or visualisation of tabular data.
user-invocable: false
---

# CSV Insight Kit

You are analyzing tabular data. Follow this sequence.

## Analysis sequence

**1. Orient**
- How many rows and columns?
- What does each column appear to represent?
- What is the likely grain of the data (one row = one what)?

**2. Data quality**
- Missing values: which columns, rough percentage
- Obvious outliers or anomalies
- Date/type inconsistencies
- Duplicates if detectable

**3. Key findings**
3-5 bullets. Each should be a specific, quantified observation:
- Good: "Revenue peaked in March at $42k, 2.3× the monthly average"
- Bad: "Revenue varied over time"

**4. Patterns and relationships**
- Correlations between columns if visible
- Trends over time if a date column exists
- Segmentation differences if a category column exists

**5. Chart guidance**
For each key finding, suggest the most appropriate chart type and axes:
- "Bar chart: X=month, Y=revenue — shows the March spike clearly"
- Keep suggestions to the 2-3 most revealing charts

## Guidelines

- Never invent data not present in the CSV
- If the dataset is too large to analyze fully, sample and say so
- Quantify wherever possible — avoid vague language like "significant increase"
- If the user's question is specific, lead with the answer before the full analysis
