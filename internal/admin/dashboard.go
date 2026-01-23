package admin

// dashboardHTML contains the embedded web UI dashboard.
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MindBalancer</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8f9fa;
            --bg-elevated: #ffffff;
            --bg-hover: #f1f3f4;
            --border: #e1e4e8;
            --border-light: #d0d7de;
            --text-primary: #1f2328;
            --text-secondary: #656d76;
            --text-muted: #8b949e;
            --accent-green: #1a7f37;
            --accent-green-bg: #dafbe1;
            --accent-red: #cf222e;
            --accent-red-bg: #ffebe9;
            --accent-blue: #0969da;
            --accent-purple: #8250df;
            --accent-orange: #bc4c00;
            --shadow-sm: 0 1px 2px rgba(0,0,0,0.04);
            --shadow-md: 0 3px 6px rgba(0,0,0,0.08);
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-secondary);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.5;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 32px 24px;
        }

        /* Header */
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 32px;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .logo-icon {
            width: 40px;
            height: 40px;
            background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 20px;
            box-shadow: var(--shadow-md);
        }

        .logo-text {
            font-size: 20px;
            font-weight: 700;
            color: var(--text-primary);
        }

        .header-right {
            display: flex;
            align-items: center;
            gap: 16px;
        }

        .status-badge {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 6px 12px;
            background: var(--accent-green-bg);
            border-radius: 20px;
            font-size: 13px;
            font-weight: 500;
            color: var(--accent-green);
        }

        .status-badge.offline {
            background: var(--accent-red-bg);
            color: var(--accent-red);
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: currentColor;
        }

        .btn {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 8px 16px;
            border-radius: 8px;
            font-size: 14px;
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

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
            margin-bottom: 24px;
        }

        .stat-card {
            background: var(--bg-elevated);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 20px;
            box-shadow: var(--shadow-sm);
        }

        .stat-label {
            font-size: 13px;
            color: var(--text-muted);
            font-weight: 500;
            margin-bottom: 8px;
        }

        .stat-value {
            font-size: 32px;
            font-weight: 700;
            font-family: 'JetBrains Mono', monospace;
            color: var(--text-primary);
        }

        .stat-subtitle {
            font-size: 12px;
            color: var(--text-muted);
            margin-top: 4px;
        }

        /* Main Grid */
        .main-grid {
            display: grid;
            grid-template-columns: 1.5fr 1fr;
            gap: 24px;
        }

        /* Card */
        .card {
            background: var(--bg-elevated);
            border: 1px solid var(--border);
            border-radius: 12px;
            box-shadow: var(--shadow-sm);
            overflow: hidden;
        }

        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 20px;
            border-bottom: 1px solid var(--border);
            background: var(--bg-secondary);
        }

        .card-title {
            font-size: 15px;
            font-weight: 600;
            color: var(--text-primary);
        }

        .card-badge {
            font-size: 12px;
            padding: 4px 10px;
            border-radius: 6px;
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
            padding: 12px 20px;
            font-size: 12px;
            font-weight: 600;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            border-bottom: 1px solid var(--border);
        }

        .table td {
            padding: 16px 20px;
            font-size: 14px;
            border-bottom: 1px solid var(--border);
        }

        .table tr:last-child td {
            border-bottom: none;
        }

        .table tr:hover {
            background: var(--bg-secondary);
        }

        .server-cell {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .server-status {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            flex-shrink: 0;
        }

        .server-status.healthy {
            background: var(--accent-green);
        }

        .server-status.unhealthy {
            background: var(--accent-red);
        }

        .server-info {
            display: flex;
            flex-direction: column;
        }

        .server-name {
            font-weight: 600;
            color: var(--text-primary);
        }

        .server-meta {
            font-size: 12px;
            color: var(--text-muted);
        }

        .badge {
            display: inline-flex;
            padding: 3px 8px;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 500;
        }

        .badge-openai { background: #d1fae5; color: #065f46; }
        .badge-anthropic { background: #ede9fe; color: #5b21b6; }
        .badge-ollama { background: #dbeafe; color: #1e40af; }
        .badge-groq { background: #ffedd5; color: #9a3412; }

        .mono {
            font-family: 'JetBrains Mono', monospace;
        }

        .text-muted { color: var(--text-muted); }
        .text-right { text-align: right; }

        /* Health List */
        .health-list {
            padding: 8px;
        }

        .health-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 14px 12px;
            border-radius: 8px;
            margin-bottom: 4px;
            transition: background 0.15s ease;
        }

        .health-item:hover {
            background: var(--bg-secondary);
        }

        .health-item:last-child {
            margin-bottom: 0;
        }

        .health-left {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .health-name {
            font-weight: 500;
            color: var(--text-primary);
        }

        .health-badge {
            font-size: 11px;
            padding: 2px 8px;
            border-radius: 4px;
            font-weight: 600;
        }

        .health-badge.healthy {
            background: var(--accent-green-bg);
            color: var(--accent-green);
        }

        .health-badge.unhealthy {
            background: var(--accent-red-bg);
            color: var(--accent-red);
        }

        .health-latency {
            font-size: 18px;
            font-weight: 600;
            font-family: 'JetBrains Mono', monospace;
            color: var(--text-primary);
        }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 48px 24px;
            color: var(--text-muted);
        }

        /* Responsive */
        @media (max-width: 1024px) {
            .stats-grid {
                grid-template-columns: repeat(2, 1fr);
            }
            .main-grid {
                grid-template-columns: 1fr;
            }
        }

        @media (max-width: 640px) {
            .stats-grid {
                grid-template-columns: 1fr;
            }
            .header {
                flex-direction: column;
                gap: 16px;
                align-items: flex-start;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Header -->
        <header class="header">
            <div class="logo">
                <div class="logo-icon">⚡</div>
                <span class="logo-text">MindBalancer</span>
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
            </div>
        </header>

        <!-- Stats -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Total Servers</div>
                <div class="stat-value" id="totalServers">-</div>
                <div class="stat-subtitle">Configured providers</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Healthy Servers</div>
                <div class="stat-value" id="healthyServers">-</div>
                <div class="stat-subtitle">Passing health checks</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Total Requests</div>
                <div class="stat-value" id="totalRequests">-</div>
                <div class="stat-subtitle">Since startup</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Avg Latency</div>
                <div class="stat-value" id="avgLatency">-</div>
                <div class="stat-subtitle">Response time</div>
            </div>
        </div>

        <!-- Main Content -->
        <div class="main-grid">
            <!-- Servers -->
            <div class="card">
                <div class="card-header">
                    <span class="card-title">Servers</span>
                    <span class="card-badge" id="algorithmBadge">weighted_round_robin</span>
                </div>
                <table class="table">
                    <thead>
                        <tr>
                            <th>Server</th>
                            <th>Provider</th>
                            <th>HG</th>
                            <th class="text-right">Requests</th>
                            <th class="text-right">Latency</th>
                        </tr>
                    </thead>
                    <tbody id="serversList">
                        <tr><td colspan="5" class="empty-state">Loading...</td></tr>
                    </tbody>
                </table>
            </div>

            <!-- Health -->
            <div class="card">
                <div class="card-header">
                    <span class="card-title">Health Status</span>
                </div>
                <div class="health-list" id="healthList">
                    <div class="empty-state">Loading...</div>
                </div>
            </div>
        </div>
    </div>

    <script>
        async function fetchStats() {
            try {
                var response = await fetch('/api/stats');
                var data = await response.json();
                
                document.getElementById('totalServers').textContent = data.TotalServers || 0;
                document.getElementById('healthyServers').textContent = data.HealthyServers || 0;
                document.getElementById('algorithmBadge').textContent = data.Algorithm || 'unknown';
                
                var totalReqs = 0;
                var totalLatency = 0;
                var latencyCount = 0;
                
                if (data.ServerStats) {
                    data.ServerStats.forEach(function(s) {
                        totalReqs += s.TotalReqs || 0;
                        if (s.AvgLatency > 0) {
                            totalLatency += s.AvgLatency;
                            latencyCount++;
                        }
                    });
                }
                
                document.getElementById('totalRequests').textContent = totalReqs.toLocaleString();
                
                if (latencyCount > 0) {
                    var avgMs = Math.round(totalLatency / latencyCount / 1000000);
                    document.getElementById('avgLatency').textContent = avgMs + ' ms';
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
        
        function renderServers(servers) {
            var container = document.getElementById('serversList');
            
            if (!servers || servers.length === 0) {
                container.innerHTML = '<tr><td colspan="5" class="empty-state">No servers configured</td></tr>';
                return;
            }
            
            container.innerHTML = servers.map(function(server) {
                var isHealthy = server.Errors === 0 || server.TotalReqs === 0;
                var latencyMs = server.AvgLatency > 0 ? Math.round(server.AvgLatency / 1000000) : 0;
                var latencyDisplay = latencyMs > 0 ? latencyMs + ' ms' : '-';
                var providerClass = getProviderClass(server.Name);
                var providerName = getProviderName(server.Name);
                
                return '<tr>' +
                    '<td>' +
                        '<div class="server-cell">' +
                            '<div class="server-status ' + (isHealthy ? 'healthy' : 'unhealthy') + '"></div>' +
                            '<div class="server-info">' +
                                '<span class="server-name">' + server.Name + '</span>' +
                                '<span class="server-meta">Weight ' + server.Weight + '</span>' +
                            '</div>' +
                        '</div>' +
                    '</td>' +
                    '<td><span class="badge ' + providerClass + '">' + providerName + '</span></td>' +
                    '<td class="mono text-muted">' + server.Hostgroup + '</td>' +
                    '<td class="text-right mono">' + server.TotalReqs.toLocaleString() + '</td>' +
                    '<td class="text-right mono">' + latencyDisplay + '</td>' +
                '</tr>';
            }).join('');
        }
        
        function renderHealth(healthData) {
            var container = document.getElementById('healthList');
            var entries = Object.entries(healthData);
            
            if (entries.length === 0) {
                container.innerHTML = '<div class="empty-state">No health data</div>';
                return;
            }
            
            container.innerHTML = entries.map(function(entry) {
                var name = entry[0];
                var status = entry[1];
                var latencyMs = status.Latency > 0 ? Math.round(status.Latency / 1000000) : 0;
                var latencyDisplay = latencyMs > 0 ? latencyMs + ' ms' : '-';
                
                return '<div class="health-item">' +
                    '<div class="health-left">' +
                        '<div class="server-status ' + (status.Healthy ? 'healthy' : 'unhealthy') + '"></div>' +
                        '<span class="health-name">' + name + '</span>' +
                        '<span class="health-badge ' + (status.Healthy ? 'healthy' : 'unhealthy') + '">' + 
                            (status.Healthy ? 'Healthy' : 'Unhealthy') + 
                        '</span>' +
                    '</div>' +
                    '<div class="health-latency">' + latencyDisplay + '</div>' +
                '</div>';
            }).join('');
        }
        
        function getProviderClass(name) {
            if (name.indexOf('openai') >= 0) return 'badge-openai';
            if (name.indexOf('anthropic') >= 0 || name.indexOf('claude') >= 0) return 'badge-anthropic';
            if (name.indexOf('ollama') >= 0) return 'badge-ollama';
            if (name.indexOf('groq') >= 0) return 'badge-groq';
            return 'badge-openai';
        }
        
        function getProviderName(name) {
            if (name.indexOf('openai') >= 0) return 'OpenAI';
            if (name.indexOf('anthropic') >= 0 || name.indexOf('claude') >= 0) return 'Anthropic';
            if (name.indexOf('ollama') >= 0) return 'Ollama';
            if (name.indexOf('azure') >= 0) return 'Azure';
            if (name.indexOf('groq') >= 0) return 'Groq';
            return 'Custom';
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
            await Promise.all([fetchStats(), fetchHealth()]);
        }
        
        refreshAll();
        setInterval(refreshAll, 5000);
    </script>
</body>
</html>`
