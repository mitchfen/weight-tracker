let chart = null;
let allWeights = [];

const MOBILE_BREAKPOINT = 600;
const DEFAULT_HISTORY_DAYS = 30;
const MILLISECONDS_PER_DAY = 24 * 60 * 60 * 1000;
let historyDays = DEFAULT_HISTORY_DAYS;

function isMobileViewport() {
    return window.innerWidth <= MOBILE_BREAKPOINT;
}

function getUtcCalendarTimestamp(recordedAt) {
    const date = new Date(recordedAt);
    return Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate());
}

function getDisplayWeights() {
    if (allWeights.length === 0) {
        return allWeights;
    }

    const newestDay = getUtcCalendarTimestamp(allWeights[allWeights.length - 1].recorded_at);
    const cutoffDay = newestDay - (historyDays - 1) * MILLISECONDS_PER_DAY;
    return allWeights.filter(weight => getUtcCalendarTimestamp(weight.recorded_at) >= cutoffDay);
}

function updateChartSummary(latestEma) {
    const summary = document.getElementById('chart-summary');
    const estimate = document.getElementById('estimated-weight');

    if (summary) summary.style.display = allWeights.length > 0 ? 'flex' : 'none';
    if (estimate) estimate.textContent = latestEma.toFixed(1);
}

function applyHistoryRange(event) {
    event.preventDefault();
    const input = document.getElementById('history-days');

    if (!input.reportValidity()) return;

    historyDays = input.valueAsNumber;
    renderChart();
}

function calculateExponentialMovingAverage(data, period = 7) {
    if (data.length === 0) return [];
    
    const multiplier = 2 / (period + 1);
    const ema = [data[0]];
    
    for (let i = 1; i < data.length; i++) {
        ema.push(data[i] * multiplier + ema[i - 1] * (1 - multiplier));
    }
    
    return ema;
}

function renderChart() {
    const weights = getDisplayWeights();
    const emptyState = document.getElementById('chart-empty-state');
    const canvas = document.getElementById('weightChart');
    const summary = document.getElementById('chart-summary');

    if (!weights || weights.length === 0) {
        if (chart) {
            chart.destroy();
            chart = null;
        }
        if (emptyState) emptyState.style.display = 'block';
        if (canvas) canvas.style.display = 'none';
        if (summary) summary.style.display = 'none';
        return;
    }

    if (emptyState) emptyState.style.display = 'none';
    if (canvas) canvas.style.display = 'block';

    const allData = allWeights.map(w => w.weight);
    const allEmaData = calculateExponentialMovingAverage(allData);

    const labels = weights.map(w => new Date(w.recorded_at).toLocaleDateString());
    const data = weights.map(w => w.weight);
    
    const firstDisplayedIndex = allWeights.indexOf(weights[0]);
    const emaData = allEmaData.slice(firstDisplayedIndex);

    const ctx = document.getElementById('weightChart').getContext('2d');

    if (chart) {
        chart.destroy();
    }

    const mobileViewport = isMobileViewport();

    chart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Weight (lbs)',
                data: data,
                borderColor: '#94a3b8',
                backgroundColor: 'transparent',
                borderWidth: 2,
                borderDash: [5, 5],
                fill: false,
                tension: 0.1,
                pointRadius: mobileViewport ? 2 : 5,
                pointBackgroundColor: '#94a3b8',
                pointBorderColor: '#fff',
                pointBorderWidth: mobileViewport ? 1 : 2,
                pointHoverRadius: mobileViewport ? 4 : 7
            }, {
                label: 'Trend (7-day EMA)',
                data: emaData,
                borderColor: '#475569',
                backgroundColor: 'rgba(71, 85, 105, 0.08)',
                borderWidth: 3,
                fill: true,
                tension: 0.3,
                pointRadius: 0,
                pointHoverRadius: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: true,
            plugins: {
                legend: {
                    display: true,
                    position: 'top',
                    labels: {
                        boxWidth: mobileViewport ? 10 : 40,
                        usePointStyle: mobileViewport
                    }
                }
            },
            scales: {
                x: {
                    ticks: {
                        maxTicksLimit: mobileViewport ? 5 : 12,
                        maxRotation: mobileViewport ? 0 : 45
                    }
                },
                y: {
                    min: Math.floor(Math.min(...data) - 1),
                    max: Math.ceil(Math.max(...data) + 1),
                    ticks: {
                        callback: function(value) {
                            return value.toFixed(1) + ' lbs';
                        },
                        maxTicksLimit: mobileViewport ? 6 : 10
                    }
                }
            }
        }
    });

    updateChartSummary(allEmaData[allEmaData.length - 1]);
}

async function loadChart() {
    try {
        const response = await fetch('/api/weights');
        allWeights = await response.json();
        renderChart();
    } catch (error) {
        console.error('Failed to load chart:', error);
    }
}

window.addEventListener('resize', renderChart);

// Load chart on page load
document.addEventListener('DOMContentLoaded', loadChart);

// Reload chart on form submission
document.addEventListener('DOMContentLoaded', function() {
    document.querySelector('form').addEventListener('htmx:afterRequest', function(event) {
        const messageContainer = document.getElementById('message-container');
        
        if (event.detail.xhr.status === 200) {
            messageContainer.innerHTML = '<div class="message success">✓ Weight recorded successfully!</div>';
            document.getElementById('weight').value = '';
            setTimeout(() => {
                messageContainer.innerHTML = '';
                loadChart();
            }, 1500);
        } else {
            messageContainer.innerHTML = '<div class="message error">✗ Failed to record weight</div>';
            setTimeout(() => {
                messageContainer.innerHTML = '';
            }, 3000);
        }
    });
});

async function importCSV(input) {
    const messageContainer = document.getElementById('message-container');
    const formData = new FormData();
    formData.append('file', input.files[0]);

    try {
        const response = await fetch('/api/weights/import', { method: 'POST', body: formData });
        const result = await response.json();
        messageContainer.innerHTML = `<div class="message success">✓ Imported ${result.imported} records</div>`;
        input.value = '';
        setTimeout(() => { messageContainer.innerHTML = ''; loadChart(); }, 1500);
    } catch (error) {
        messageContainer.innerHTML = '<div class="message error">✗ Failed to import CSV</div>';
        setTimeout(() => { messageContainer.innerHTML = ''; }, 3000);
    }
}
