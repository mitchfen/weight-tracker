#!/usr/bin/env python3
"""
Generate realistic test weight data for the weight tracker app.
Populates weights.db with 50 days of weight entries between 190-198 lbs.
"""

import sqlite3
from datetime import datetime, timedelta
import random
import sys
import os

db_path = os.path.join(os.path.dirname(__file__), "..", "weights.db")
conn = sqlite3.connect(db_path)
cursor = conn.cursor()

# Start 50 days ago
start_date = datetime.now() - timedelta(days=49)
current_weight = 194.0  # Starting weight

weights_data = []

for i in range(50):
    # Small realistic change: -0.5 to +0.5 lbs per day
    change = random.uniform(-0.5, 0.5)
    current_weight = max(190, min(198, current_weight + change))
    
    recorded_date = (start_date + timedelta(days=i)).strftime("%Y-%m-%d")
    weights_data.append((current_weight, recorded_date))

# Insert data
for weight, date in weights_data:
    cursor.execute(
        "INSERT INTO weights (weight, recorded_date) VALUES (?, ?) ON CONFLICT(recorded_date) DO UPDATE SET weight=excluded.weight",
        (weight, date)
    )

conn.commit()
conn.close()
print(f"✓ Inserted {len(weights_data)} weight records")
print(f"  Range: {min(w[0] for w in weights_data):.1f} - {max(w[0] for w in weights_data):.1f} lbs")
