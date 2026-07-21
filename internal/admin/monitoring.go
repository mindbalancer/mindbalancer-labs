package admin

// monitoringHTML contains the embedded monitoring dashboard.
// This is a public page that does not require authentication.
const monitoringHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MindBalancer - Monitoring</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #0d1117;
            --bg-secondary: #161b22;
            --bg-elevated: #21262d;
            --bg-hover: #30363d;
            --border: #30363d;
            --border-light: #484f58;
            --text-primary: #f0f6fc;
            --text-secondary: #8b949e;
            --text-muted: #6e7681;
            --accent-green: #3fb950;
            --accent-green-bg: rgba(63, 185, 80, 0.15);
            --accent-red: #f85149;
            --accent-red-bg: rgba(248, 81, 73, 0.15);
            --accent-blue: #58a6ff;
            --accent-purple: #a371f7;
            --accent-orange: #d29922;
            --accent-cyan: #39c5cf;
            --shadow-sm: 0 1px 2px rgba(0,0,0,0.3);
            --shadow-md: 0 3px 8px rgba(0,0,0,0.4);
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.5;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 24px;
        }

        /* Header */
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 24px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border);
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .logo-icon {
            width: 36px;
            height: 36px;
            border-radius: 8px;
            object-fit: contain;
        }

        .logo-text {
            font-size: 18px;
            font-weight: 700;
        }

        .logo-badge {
            font-size: 11px;
            padding: 2px 8px;
            background: var(--accent-blue);
            color: var(--bg-primary);
            border-radius: 4px;
            font-weight: 600;
            margin-left: 8px;
        }

        .header-right {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .status-badge {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 6px 12px;
            background: var(--accent-green-bg);
            border: 1px solid var(--accent-green);
            border-radius: 6px;
            font-size: 12px;
            font-weight: 500;
            color: var(--accent-green);
        }

        .status-badge.offline {
            background: var(--accent-red-bg);
            border-color: var(--accent-red);
            color: var(--accent-red);
        }

        .status-dot {
            width: 6px;
            height: 6px;
            border-radius: 50%;
            background: currentColor;
            animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        .btn {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 8px 14px;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.15s ease;
            border: 1px solid var(--border);
            background: var(--bg-elevated);
            color: var(--text-secondary);
            font-family: inherit;
        }

        .btn:hover {
            background: var(--bg-hover);
            border-color: var(--border-light);
            color: var(--text-primary);
        }

        .btn-admin {
            background: var(--accent-purple);
            border-color: var(--accent-purple);
            color: white;
        }

        .btn-admin:hover {
            background: #8250df;
            border-color: #8250df;
            color: white;
        }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
            margin-bottom: 24px;
        }

        .stat-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 20px;
        }

        .stat-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 8px;
        }

        .stat-label {
            font-size: 12px;
            color: var(--text-muted);
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .stat-icon {
            width: 32px;
            height: 32px;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 16px;
        }

        .stat-icon.blue { background: rgba(88, 166, 255, 0.15); color: var(--accent-blue); }
        .stat-icon.green { background: rgba(63, 185, 80, 0.15); color: var(--accent-green); }
        .stat-icon.orange { background: rgba(210, 153, 34, 0.15); color: var(--accent-orange); }
        .stat-icon.purple { background: rgba(163, 113, 247, 0.15); color: var(--accent-purple); }

        .stat-value {
            font-size: 28px;
            font-weight: 700;
            font-family: 'JetBrains Mono', monospace;
            margin-bottom: 4px;
        }

        .stat-subtitle {
            font-size: 12px;
            color: var(--text-muted);
        }

        .stat-change {
            font-size: 11px;
            padding: 2px 6px;
            border-radius: 4px;
            font-weight: 500;
        }

        .stat-change.positive { background: var(--accent-green-bg); color: var(--accent-green); }
        .stat-change.negative { background: var(--accent-red-bg); color: var(--accent-red); }

        /* Main Grid */
        .main-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 24px;
            margin-bottom: 24px;
        }

        /* Cache Grid */
        .cache-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
            margin-bottom: 24px;
        }

        .cache-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 10px;
            padding: 16px;
            text-align: center;
        }

        .cache-value {
            font-size: 24px;
            font-weight: 700;
            font-family: 'JetBrains Mono', monospace;
            color: var(--accent-cyan);
        }

        .cache-label {
            font-size: 11px;
            color: var(--text-muted);
            text-transform: uppercase;
            margin-top: 4px;
        }

        /* Card */
        .card {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 12px;
            overflow: hidden;
        }

        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 20px;
            border-bottom: 1px solid var(--border);
        }

        .card-title {
            font-size: 14px;
            font-weight: 600;
        }

        .card-badge {
            font-size: 11px;
            padding: 4px 8px;
            border-radius: 4px;
            background: var(--bg-hover);
            color: var(--text-muted);
            font-family: 'JetBrains Mono', monospace;
        }

        /* Table */
        .table {
            width: 100%;
            border-collapse: collapse;
        }

        .table th {
            text-align: left;
            padding: 12px 16px;
            font-size: 11px;
            font-weight: 600;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            border-bottom: 1px solid var(--border);
            background: var(--bg-elevated);
        }

        .table td {
            padding: 14px 16px;
            font-size: 13px;
            border-bottom: 1px solid var(--border);
        }

        .table tr:last-child td {
            border-bottom: none;
        }

        .table tr:hover {
            background: var(--bg-elevated);
        }

        .server-cell {
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .server-status {
            width: 8px;
            height: 8px;
            border-radius: 50%;
        }

        .server-status.healthy { background: var(--accent-green); box-shadow: 0 0 8px var(--accent-green); }
        .server-status.unhealthy { background: var(--accent-red); box-shadow: 0 0 8px var(--accent-red); }

        .server-name {
            font-weight: 500;
        }

        .badge {
            display: inline-flex;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 500;
        }

        .badge-openai { background: rgba(0, 163, 108, 0.2); color: #00d68f; }
        .badge-anthropic { background: rgba(163, 113, 247, 0.2); color: #a371f7; }
        .badge-ollama { background: rgba(88, 166, 255, 0.2); color: #58a6ff; }
        .badge-groq { background: rgba(255, 123, 0, 0.2); color: #ff9f43; }
        .badge-google { background: rgba(66, 133, 244, 0.2); color: #4285f4; }
        .badge-azure { background: rgba(0, 127, 255, 0.2); color: #007fff; }

        .mono {
            font-family: 'JetBrains Mono', monospace;
        }

        .text-muted { color: var(--text-muted); }
        .text-right { text-align: right; }
        .text-green { color: var(--accent-green); }
        .text-red { color: var(--accent-red); }

        /* Health Grid */
        .health-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 12px;
            padding: 16px;
        }

        .health-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px;
            background: var(--bg-elevated);
            border-radius: 8px;
            border: 1px solid var(--border);
        }

        .health-info {
            flex: 1;
            min-width: 0;
        }

        .health-name {
            font-size: 13px;
            font-weight: 500;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .health-meta {
            font-size: 11px;
            color: var(--text-muted);
        }

        .health-latency {
            font-size: 14px;
            font-weight: 600;
            font-family: 'JetBrains Mono', monospace;
            color: var(--accent-cyan);
        }

        /* Request Log */
        .log-list {
            max-height: 300px;
            overflow-y: auto;
        }

        .log-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 10px 16px;
            border-bottom: 1px solid var(--border);
            font-size: 12px;
        }

        .log-item:last-child {
            border-bottom: none;
        }

        .log-time {
            color: var(--text-muted);
            font-family: 'JetBrains Mono', monospace;
            flex-shrink: 0;
        }

        .log-server {
            font-weight: 500;
            flex-shrink: 0;
            min-width: 100px;
        }

        .log-model {
            color: var(--text-secondary);
            flex: 1;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .log-status {
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 11px;
            font-weight: 600;
            flex-shrink: 0;
        }

        .log-status.success { background: var(--accent-green-bg); color: var(--accent-green); }
        .log-status.error { background: var(--accent-red-bg); color: var(--accent-red); }

        .log-latency {
            font-family: 'JetBrains Mono', monospace;
            color: var(--text-muted);
            flex-shrink: 0;
            min-width: 60px;
            text-align: right;
        }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 40px 24px;
            color: var(--text-muted);
        }

        /* Last Updated */
        .last-updated {
            text-align: center;
            padding: 16px;
            font-size: 12px;
            color: var(--text-muted);
            border-top: 1px solid var(--border);
        }

        /* Responsive */
        @media (max-width: 1200px) {
            .stats-grid { grid-template-columns: repeat(2, 1fr); }
            .cache-grid { grid-template-columns: repeat(2, 1fr); }
            .main-grid { grid-template-columns: 1fr; }
            .health-grid { grid-template-columns: repeat(2, 1fr); }
        }

        @media (max-width: 640px) {
            .stats-grid { grid-template-columns: 1fr; }
            .cache-grid { grid-template-columns: 1fr; }
            .health-grid { grid-template-columns: 1fr; }
            .header { flex-direction: column; gap: 12px; align-items: flex-start; }
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Header -->
        <header class="header">
            <div class="logo">
                <img src="/static/logo-mindbalancer.png" alt="MindBalancer" class="logo-icon">
                <span class="logo-text">MindBalancer</span>
                <span class="logo-badge">MONITORING</span>
            </div>
            <div class="header-right">
                <div class="status-badge" id="connectionStatus">
                    <span class="status-dot"></span>
                    <span>Connected</span>
                </div>
                <button class="btn" onclick="refreshAll()">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                    </svg>
                    Refresh
                </button>
                <a href="/admin" class="btn btn-admin">Admin Panel</a>
            </div>
        </header>

        <!-- Stats -->
        <section class="stats-grid">
            <div class="stat-card">
                <div class="stat-header">
                    <span class="stat-label">Total Requests</span>
                    <div class="stat-icon blue"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg></div>
                </div>
                <div class="stat-value" id="totalRequests">-</div>
                <div class="stat-subtitle">Since startup</div>
            </div>
            <div class="stat-card">
                <div class="stat-header">
                    <span class="stat-label">Success Rate</span>
                    <div class="stat-icon green"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg></div>
                </div>
                <div class="stat-value" id="successRate">-</div>
                <div class="stat-subtitle">Successful requests</div>
            </div>
            <div class="stat-card">
                <div class="stat-header">
                    <span class="stat-label">Avg Latency</span>
                    <div class="stat-icon orange"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg></div>
                </div>
                <div class="stat-value" id="avgLatency">-</div>
                <div class="stat-subtitle">Response time</div>
            </div>
            <div class="stat-card">
                <div class="stat-header">
                    <span class="stat-label">Healthy Servers</span>
                    <div class="stat-icon purple"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1" fill="currentColor"/><circle cx="6" cy="18" r="1" fill="currentColor"/></svg></div>
                </div>
                <div class="stat-value" id="healthyServers">-</div>
                <div class="stat-subtitle" id="totalServersSubtitle">of 0 total</div>
            </div>
        </section>

        <!-- Cache Stats -->
        <section class="cache-grid">
            <div class="cache-card">
                <div class="cache-value" id="cacheHitRate">-</div>
                <div class="cache-label">Cache Hit Rate</div>
            </div>
            <div class="cache-card">
                <div class="cache-value" id="cacheHits">-</div>
                <div class="cache-label">Cache Hits</div>
            </div>
            <div class="cache-card">
                <div class="cache-value" id="cacheMemory">-</div>
                <div class="cache-label">Memory Used</div>
            </div>
            <div class="cache-card">
                <div class="cache-value" id="cacheItems">-</div>
                <div class="cache-label">Cached Items</div>
            </div>
        </section>

        <!-- Main Content -->
        <section class="main-grid">
            <!-- Servers -->
            <div class="card">
                <div class="card-header">
                    <span class="card-title">Server Performance</span>
                    <span class="card-badge" id="algorithmBadge">-</span>
                </div>
                <table class="table">
                    <thead>
                        <tr>
                            <th>Server</th>
                            <th>Provider</th>
                            <th>HG</th>
                            <th class="text-right">Requests</th>
                            <th class="text-right">Errors</th>
                            <th class="text-right">Latency</th>
                        </tr>
                    </thead>
                    <tbody id="serversList">
                        <tr><td colspan="6" class="empty-state">Loading...</td></tr>
                    </tbody>
                </table>
            </div>

            <!-- Recent Requests -->
            <div class="card">
                <div class="card-header">
                    <span class="card-title">Recent Requests</span>
                    <span class="card-badge" id="requestCount">0</span>
                </div>
                <div class="log-list" id="requestLog">
                    <div class="empty-state">Loading...</div>
                </div>
            </div>
        </section>

        <!-- Health Status -->
        <div class="card">
            <div class="card-header">
                <span class="card-title">Health Status</span>
            </div>
            <div class="health-grid" id="healthGrid">
                <div class="empty-state">Loading...</div>
            </div>
        </div>

        <!-- Last Updated -->
        <div class="last-updated">
            Last updated: <span id="lastUpdated">-</span> | Auto-refresh every 5 seconds
        </div>
    </div>

    <script>
        var totalErrors = 0;
        var totalReqs = 0;

        async function fetchStats() {
            try {
                var response = await fetch('/api/stats');
                var data = await response.json();
                
                document.getElementById('healthyServers').textContent = data.HealthyServers || 0;
                document.getElementById('totalServersSubtitle').textContent = 'of ' + (data.TotalServers || 0) + ' total';
                document.getElementById('algorithmBadge').textContent = data.Algorithm || 'unknown';
                
                totalReqs = 0;
                totalErrors = 0;
                var totalLatency = 0;
                var latencyCount = 0;
                
                if (data.ServerStats) {
                    data.ServerStats.forEach(function(s) {
                        totalReqs += s.TotalReqs || 0;
                        totalErrors += s.Errors || 0;
                        if (s.AvgLatency > 0) {
                            totalLatency += s.AvgLatency;
                            latencyCount++;
                        }
                    });
                }
                
                document.getElementById('totalRequests').textContent = formatNumber(totalReqs);
                
                var successRate = totalReqs > 0 ? ((totalReqs - totalErrors) / totalReqs * 100).toFixed(1) + '%' : '-';
                document.getElementById('successRate').textContent = successRate;
                
                if (latencyCount > 0) {
                    var avgMs = Math.round(totalLatency / latencyCount / 1000000);
                    document.getElementById('avgLatency').textContent = avgMs + 'ms';
                } else {
                    document.getElementById('avgLatency').textContent = '-';
                }
                
                renderServers(data.ServerStats || []);
                setOnline();
                
            } catch (error) {
                console.error('Failed to fetch stats:', error);
                setOffline();
            }
        }
        
        async function fetchHealth() {
            try {
                var response = await fetch('/api/health');
                var data = await response.json();
                renderHealth(data);
            } catch (error) {
                console.error('Failed to fetch health:', error);
            }
        }

        async function fetchCache() {
            try {
                var response = await fetch('/api/cache');
                var data = await response.json();
                
                var hitRate = data.hit_rate !== undefined ? (data.hit_rate * 100).toFixed(1) + '%' : '-';
                document.getElementById('cacheHitRate').textContent = hitRate;
                document.getElementById('cacheHits').textContent = formatNumber(data.hits || 0);
                document.getElementById('cacheMemory').textContent = formatBytes(data.memory_used_bytes || 0);
                document.getElementById('cacheItems').textContent = formatNumber(data.item_count || 0);
            } catch (error) {
                console.error('Failed to fetch cache:', error);
            }
        }

        async function fetchRequestLog() {
            try {
                var response = await fetch('/api/stats/requests?limit=20');
                var data = await response.json();
                renderRequestLog(data || []);
                document.getElementById('requestCount').textContent = data ? data.length : 0;
            } catch (error) {
                console.error('Failed to fetch request log:', error);
            }
        }
        
        function renderServers(servers) {
            var container = document.getElementById('serversList');
            
            if (!servers || servers.length === 0) {
                container.innerHTML = '<tr><td colspan="6" class="empty-state">No servers configured</td></tr>';
                return;
            }
            
            container.innerHTML = servers.map(function(server) {
                var isHealthy = server.Errors === 0 || server.TotalReqs === 0;
                var latencyMs = server.AvgLatency > 0 ? Math.round(server.AvgLatency / 1000000) : 0;
                var latencyDisplay = latencyMs > 0 ? latencyMs + 'ms' : '-';
                var providerClass = getProviderClass(server.Name);
                var providerName = getProviderName(server.Name);
                var errorClass = server.Errors > 0 ? 'text-red' : 'text-muted';
                
                return '<tr>' +
                    '<td>' +
                        '<div class="server-cell">' +
                            '<div class="server-status ' + (isHealthy ? 'healthy' : 'unhealthy') + '"></div>' +
                            '<span class="server-name">' + escapeHtml(server.Name) + '</span>' +
                        '</div>' +
                    '</td>' +
                    '<td><span class="badge ' + providerClass + '">' + providerName + '</span></td>' +
                    '<td class="mono text-muted">' + server.Hostgroup + '</td>' +
                    '<td class="text-right mono">' + formatNumber(server.TotalReqs) + '</td>' +
                    '<td class="text-right mono ' + errorClass + '">' + server.Errors + '</td>' +
                    '<td class="text-right mono">' + latencyDisplay + '</td>' +
                '</tr>';
            }).join('');
        }
        
        function renderHealth(healthData) {
            var container = document.getElementById('healthGrid');
            var entries = Object.entries(healthData);
            
            if (entries.length === 0) {
                container.innerHTML = '<div class="empty-state">No health data available</div>';
                return;
            }
            
            container.innerHTML = entries.map(function(entry) {
                var name = entry[0];
                var status = entry[1];
                var latencyMs = status.Latency > 0 ? Math.round(status.Latency / 1000000) : 0;
                var latencyDisplay = latencyMs > 0 ? latencyMs + 'ms' : '-';
                
                return '<div class="health-item">' +
                    '<div class="server-status ' + (status.Healthy ? 'healthy' : 'unhealthy') + '"></div>' +
                    '<div class="health-info">' +
                        '<div class="health-name">' + escapeHtml(name) + '</div>' +
                        '<div class="health-meta">' + (status.Healthy ? 'Healthy' : 'Unhealthy') + '</div>' +
                    '</div>' +
                    '<div class="health-latency">' + latencyDisplay + '</div>' +
                '</div>';
            }).join('');
        }

        function renderRequestLog(logs) {
            var container = document.getElementById('requestLog');
            
            if (!logs || logs.length === 0) {
                container.innerHTML = '<div class="empty-state">No recent requests</div>';
                return;
            }
            
            container.innerHTML = logs.map(function(log) {
                var isSuccess = log.StatusCode >= 200 && log.StatusCode < 400;
                var time = new Date(log.Timestamp).toLocaleTimeString();
                
                return '<div class="log-item">' +
                    '<span class="log-time">' + time + '</span>' +
                    '<span class="log-server">' + escapeHtml(log.ServerName || '-') + '</span>' +
                    '<span class="log-model">' + escapeHtml(log.Model || '-') + '</span>' +
                    '<span class="log-status ' + (isSuccess ? 'success' : 'error') + '">' + log.StatusCode + '</span>' +
                    '<span class="log-latency">' + log.LatencyMS + 'ms</span>' +
                '</div>';
            }).join('');
        }
        
        function escapeHtml(value) {
            return String(value == null ? '' : value)
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&#39;');
        }

        function getProviderClass(name) {
            name = name.toLowerCase();
            if (name.indexOf('openai') >= 0) return 'badge-openai';
            if (name.indexOf('anthropic') >= 0 || name.indexOf('claude') >= 0) return 'badge-anthropic';
            if (name.indexOf('ollama') >= 0) return 'badge-ollama';
            if (name.indexOf('groq') >= 0) return 'badge-groq';
            if (name.indexOf('google') >= 0 || name.indexOf('gemini') >= 0) return 'badge-google';
            if (name.indexOf('azure') >= 0) return 'badge-azure';
            return 'badge-openai';
        }
        
        function getProviderName(name) {
            name = name.toLowerCase();
            if (name.indexOf('openai') >= 0) return 'OpenAI';
            if (name.indexOf('anthropic') >= 0 || name.indexOf('claude') >= 0) return 'Anthropic';
            if (name.indexOf('ollama') >= 0) return 'Ollama';
            if (name.indexOf('groq') >= 0) return 'Groq';
            if (name.indexOf('google') >= 0 || name.indexOf('gemini') >= 0) return 'Google';
            if (name.indexOf('azure') >= 0) return 'Azure';
            return 'Custom';
        }

        function formatNumber(num) {
            if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
            if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
            return num.toString();
        }

        function formatBytes(bytes) {
            if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB';
            if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB';
            if (bytes >= 1024) return (bytes / 1024).toFixed(1) + ' KB';
            return bytes + ' B';
        }
        
        function setOffline() {
            var el = document.getElementById('connectionStatus');
            el.className = 'status-badge offline';
            el.innerHTML = '<span class="status-dot"></span><span>Disconnected</span>';
        }
        
        function setOnline() {
            var el = document.getElementById('connectionStatus');
            el.className = 'status-badge';
            el.innerHTML = '<span class="status-dot"></span><span>Connected</span>';
        }
        
        async function refreshAll() {
            await Promise.all([fetchStats(), fetchHealth(), fetchCache(), fetchRequestLog()]);
            document.getElementById('lastUpdated').textContent = new Date().toLocaleTimeString();
        }
        
        refreshAll();
        setInterval(refreshAll, 5000);
    </script>
</body>
</html>`
