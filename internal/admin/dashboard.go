package admin

// dashboardHTML contains the embedded web UI dashboard (Sentry.io inspired).
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MindBalancer</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Rubik:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #1a141f;
            --bg-secondary: #241c2a;
            --bg-elevated: #2e2435;
            --bg-hover: #3a2f42;
            --border: #453a4f;
            --border-light: #5a4d66;
            --text-primary: #ebe6ef;
            --text-secondary: #b4aabb;
            --text-muted: #8a8091;
            --accent-purple: #6c5fc7;
            --accent-purple-light: #8679d2;
            --accent-pink: #f55459;
            --accent-green: #2ba676;
            --accent-green-light: #45c48a;
            --accent-yellow: #f5b000;
            --accent-blue: #3b6ecc;
            --accent-orange: #f58c46;
            --gradient-purple: linear-gradient(135deg, #6c5fc7 0%, #8679d2 100%);
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Rubik', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.5;
        }

        /* Sidebar */
        .sidebar {
            position: fixed;
            left: 0;
            top: 0;
            bottom: 0;
            width: 220px;
            background: var(--bg-secondary);
            border-right: 1px solid var(--border);
            display: flex;
            flex-direction: column;
            z-index: 100;
        }

        .sidebar-header {
            padding: 20px;
            border-bottom: 1px solid var(--border);
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .logo-icon {
            width: 32px;
            height: 32px;
            background: var(--gradient-purple);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 16px;
        }

        .logo-text {
            font-size: 16px;
            font-weight: 600;
            color: var(--text-primary);
        }

        .sidebar-nav {
            padding: 12px;
            flex: 1;
        }

        .nav-section {
            margin-bottom: 24px;
        }

        .nav-section-title {
            font-size: 11px;
            font-weight: 600;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            padding: 8px 12px;
        }

        .nav-item {
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 10px 12px;
            border-radius: 6px;
            color: var(--text-secondary);
            text-decoration: none;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.15s ease;
            cursor: pointer;
        }

        .nav-item:hover {
            background: var(--bg-hover);
            color: var(--text-primary);
        }

        .nav-item.active {
            background: var(--accent-purple);
            color: white;
        }

        .nav-icon {
            width: 18px;
            height: 18px;
            opacity: 0.7;
        }

        .sidebar-footer {
            padding: 16px;
            border-top: 1px solid var(--border);
        }

        .status-indicator {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 12px;
            color: var(--text-muted);
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: var(--accent-green);
            animation: pulse 2s infinite;
        }

        .status-dot.error { background: var(--accent-pink); }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        /* Main Content */
        .main {
            margin-left: 220px;
            min-height: 100vh;
        }

        .header {
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border);
            padding: 16px 32px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: sticky;
            top: 0;
            z-index: 50;
        }

        .header-title {
            font-size: 18px;
            font-weight: 600;
        }

        .header-actions {
            display: flex;
            align-items: center;
            gap: 12px;
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
            border: none;
            font-family: inherit;
        }

        .btn-secondary {
            background: var(--bg-elevated);
            color: var(--text-secondary);
            border: 1px solid var(--border);
        }

        .btn-secondary:hover {
            background: var(--bg-hover);
            color: var(--text-primary);
            border-color: var(--border-light);
        }

        .btn-primary {
            background: var(--accent-purple);
            color: white;
        }

        .btn-primary:hover {
            background: var(--accent-purple-light);
        }

        .content {
            padding: 24px 32px;
        }

        /* Stats Row */
        .stats-row {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
            margin-bottom: 24px;
        }

        .stat-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 20px;
        }

        .stat-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 8px;
        }

        .stat-label {
            font-size: 13px;
            color: var(--text-muted);
            font-weight: 500;
        }

        .stat-trend {
            font-size: 11px;
            padding: 2px 6px;
            border-radius: 4px;
            font-weight: 500;
        }

        .stat-trend.up {
            background: rgba(43, 166, 118, 0.15);
            color: var(--accent-green-light);
        }

        .stat-trend.down {
            background: rgba(245, 84, 89, 0.15);
            color: var(--accent-pink);
        }

        .stat-value {
            font-size: 32px;
            font-weight: 700;
            font-family: 'IBM Plex Mono', monospace;
            line-height: 1.2;
        }

        .stat-subtitle {
            font-size: 12px;
            color: var(--text-muted);
            margin-top: 4px;
        }

        /* Cards */
        .card {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 8px;
            margin-bottom: 24px;
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
            background: var(--bg-elevated);
            color: var(--text-muted);
            font-family: 'IBM Plex Mono', monospace;
        }

        .card-body {
            padding: 0;
        }

        /* Table */
        .table {
            width: 100%;
            border-collapse: collapse;
        }

        .table th {
            text-align: left;
            padding: 12px 20px;
            font-size: 11px;
            font-weight: 600;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            background: var(--bg-elevated);
            border-bottom: 1px solid var(--border);
        }

        .table td {
            padding: 14px 20px;
            font-size: 13px;
            border-bottom: 1px solid var(--border);
        }

        .table tr:last-child td {
            border-bottom: none;
        }

        .table tr:hover {
            background: var(--bg-hover);
        }

        .server-name {
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .server-status {
            width: 10px;
            height: 10px;
            border-radius: 50%;
        }

        .server-status.healthy {
            background: var(--accent-green);
            box-shadow: 0 0 8px var(--accent-green);
        }

        .server-status.unhealthy {
            background: var(--accent-pink);
            box-shadow: 0 0 8px var(--accent-pink);
        }

        .server-info {
            display: flex;
            flex-direction: column;
        }

        .server-title {
            font-weight: 500;
            color: var(--text-primary);
        }

        .server-endpoint {
            font-size: 11px;
            color: var(--text-muted);
            font-family: 'IBM Plex Mono', monospace;
        }

        .badge {
            display: inline-flex;
            align-items: center;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
        }

        .badge-openai {
            background: rgba(43, 166, 118, 0.15);
            color: var(--accent-green-light);
        }

        .badge-anthropic {
            background: rgba(108, 95, 199, 0.2);
            color: var(--accent-purple-light);
        }

        .badge-ollama {
            background: rgba(59, 110, 204, 0.15);
            color: var(--accent-blue);
        }

        .badge-groq {
            background: rgba(245, 140, 70, 0.15);
            color: var(--accent-orange);
        }

        .badge-healthy {
            background: rgba(43, 166, 118, 0.15);
            color: var(--accent-green-light);
        }

        .badge-unhealthy {
            background: rgba(245, 84, 89, 0.15);
            color: var(--accent-pink);
        }

        .mono {
            font-family: 'IBM Plex Mono', monospace;
        }

        .text-muted {
            color: var(--text-muted);
        }

        .text-right {
            text-align: right;
        }

        /* Grid Layout */
        .grid-2 {
            display: grid;
            grid-template-columns: 2fr 1fr;
            gap: 24px;
        }

        /* Health List */
        .health-list {
            padding: 8px;
        }

        .health-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 12px;
            border-radius: 6px;
            margin-bottom: 4px;
        }

        .health-item:hover {
            background: var(--bg-hover);
        }

        .health-item:last-child {
            margin-bottom: 0;
        }

        .health-name {
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .health-latency {
            font-size: 18px;
            font-weight: 600;
            font-family: 'IBM Plex Mono', monospace;
        }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 48px 24px;
            color: var(--text-muted);
        }

        .empty-icon {
            font-size: 48px;
            margin-bottom: 16px;
            opacity: 0.5;
        }

        /* Responsive */
        @media (max-width: 1200px) {
            .stats-row {
                grid-template-columns: repeat(2, 1fr);
            }
            .grid-2 {
                grid-template-columns: 1fr;
            }
        }

        @media (max-width: 768px) {
            .sidebar {
                display: none;
            }
            .main {
                margin-left: 0;
            }
            .stats-row {
                grid-template-columns: 1fr;
            }
        }

        /* Scrollbar */
        ::-webkit-scrollbar { width: 8px; height: 8px; }
        ::-webkit-scrollbar-track { background: var(--bg-primary); }
        ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 4px; }
        ::-webkit-scrollbar-thumb:hover { background: var(--border-light); }
    </style>
</head>
<body>
    <!-- Sidebar -->
    <aside class="sidebar">
        <div class="sidebar-header">
            <div class="logo">
                <div class="logo-icon">⚡</div>
                <span class="logo-text">MindBalancer</span>
            </div>
        </div>
        <nav class="sidebar-nav">
            <div class="nav-section">
                <div class="nav-section-title">Overview</div>
                <a class="nav-item active">
                    <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="3" y="3" width="7" height="9" rx="1"/>
                        <rect x="14" y="3" width="7" height="5" rx="1"/>
                        <rect x="14" y="12" width="7" height="9" rx="1"/>
                        <rect x="3" y="16" width="7" height="5" rx="1"/>
                    </svg>
                    Dashboard
                </a>
                <a class="nav-item">
                    <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="2" y="3" width="20" height="14" rx="2"/>
                        <path d="M8 21h8M12 17v4"/>
                    </svg>
                    Servers
                </a>
                <a class="nav-item">
                    <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 20V10M6 20V4M18 20v-4"/>
                    </svg>
                    Metrics
                </a>
            </div>
            <div class="nav-section">
                <div class="nav-section-title">Configuration</div>
                <a class="nav-item">
                    <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="16 3 21 3 21 8"/>
                        <line x1="4" y1="20" x2="21" y2="3"/>
                        <polyline points="21 16 21 21 16 21"/>
                        <line x1="15" y1="15" x2="21" y2="21"/>
                        <line x1="4" y1="4" x2="9" y2="9"/>
                    </svg>
                    Routing
                </a>
                <a class="nav-item">
                    <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                        <circle cx="9" cy="7" r="4"/>
                        <path d="M23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"/>
                    </svg>
                    Users
                </a>
                <a class="nav-item">
                    <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="3"/>
                        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
                    </svg>
                    Settings
                </a>
            </div>
        </nav>
        <div class="sidebar-footer">
            <div class="status-indicator" id="connectionStatus">
                <span class="status-dot"></span>
                <span>Connected</span>
            </div>
        </div>
    </aside>

    <!-- Main Content -->
    <main class="main">
        <header class="header">
            <h1 class="header-title">Dashboard</h1>
            <div class="header-actions">
                <span class="text-muted" style="font-size: 12px;" id="lastUpdated">Updated just now</span>
                <button class="btn btn-secondary" onclick="refreshAll()">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                    </svg>
                    Refresh
                </button>
            </div>
        </header>

        <div class="content">
            <!-- Stats Row -->
            <div class="stats-row">
                <div class="stat-card">
                    <div class="stat-header">
                        <span class="stat-label">Total Servers</span>
                    </div>
                    <div class="stat-value" id="totalServers">-</div>
                    <div class="stat-subtitle">Configured providers</div>
                </div>
                <div class="stat-card">
                    <div class="stat-header">
                        <span class="stat-label">Healthy</span>
                        <span class="stat-trend up" id="healthTrend">100%</span>
                    </div>
                    <div class="stat-value" id="healthyServers">-</div>
                    <div class="stat-subtitle">Passing health checks</div>
                </div>
                <div class="stat-card">
                    <div class="stat-header">
                        <span class="stat-label">Total Requests</span>
                    </div>
                    <div class="stat-value" id="totalRequests">-</div>
                    <div class="stat-subtitle">Since startup</div>
                </div>
                <div class="stat-card">
                    <div class="stat-header">
                        <span class="stat-label">Avg Latency</span>
                    </div>
                    <div class="stat-value" id="avgLatency">-</div>
                    <div class="stat-subtitle">Response time</div>
                </div>
            </div>

            <!-- Grid -->
            <div class="grid-2">
                <!-- Servers Table -->
                <div class="card">
                    <div class="card-header">
                        <h2 class="card-title">Servers</h2>
                        <span class="card-badge" id="algorithmBadge">weighted_round_robin</span>
                    </div>
                    <div class="card-body">
                        <table class="table" id="serversTable">
                            <thead>
                                <tr>
                                    <th>Server</th>
                                    <th>Provider</th>
                                    <th>Hostgroup</th>
                                    <th class="text-right">Requests</th>
                                    <th class="text-right">Latency</th>
                                    <th class="text-right">Status</th>
                                </tr>
                            </thead>
                            <tbody id="serversList">
                                <tr>
                                    <td colspan="6" class="empty-state">Loading...</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <!-- Health Status -->
                <div class="card">
                    <div class="card-header">
                        <h2 class="card-title">Health Checks</h2>
                    </div>
                    <div class="card-body">
                        <div class="health-list" id="healthList">
                            <div class="empty-state">Loading...</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </main>

    <script>
        let lastUpdate = new Date();

        async function fetchStats() {
            try {
                const response = await fetch('/api/stats');
                const data = await response.json();
                
                document.getElementById('totalServers').textContent = data.TotalServers || 0;
                document.getElementById('healthyServers').textContent = data.HealthyServers || 0;
                document.getElementById('algorithmBadge').textContent = data.Algorithm || 'unknown';
                
                // Health percentage
                if (data.TotalServers > 0) {
                    const pct = Math.round((data.HealthyServers / data.TotalServers) * 100);
                    const trend = document.getElementById('healthTrend');
                    trend.textContent = pct + '%';
                    trend.className = 'stat-trend ' + (pct >= 80 ? 'up' : 'down');
                }
                
                // Calculate totals
                let totalReqs = 0;
                let totalLatency = 0;
                let latencyCount = 0;
                
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
                    document.getElementById('avgLatency').textContent = '- ms';
                }
                
                renderServers(data.ServerStats || []);
                setOnline();
                updateLastUpdated();
                
            } catch (error) {
                console.error('Failed to fetch stats:', error);
                setOffline();
            }
        }
        
        async function fetchHealth() {
            try {
                const response = await fetch('/api/health');
                const data = await response.json();
                renderHealth(data);
            } catch (error) {
                console.error('Failed to fetch health:', error);
            }
        }
        
        function renderServers(servers) {
            var container = document.getElementById('serversList');
            
            if (!servers || servers.length === 0) {
                container.innerHTML = '<tr><td colspan="6" class="empty-state"><div class="empty-icon">📡</div>No servers configured</td></tr>';
                return;
            }
            
            container.innerHTML = servers.map(function(server) {
                var isHealthy = server.Errors === 0 || server.TotalReqs === 0;
                var latencyMs = server.AvgLatency > 0 ? Math.round(server.AvgLatency / 1000000) : 0;
                var latencyDisplay = latencyMs > 0 ? latencyMs + ' ms' : '- ms';
                var providerClass = getProviderClass(server.Name);
                var providerName = getProviderName(server.Name);
                
                return '<tr>' +
                    '<td>' +
                        '<div class="server-name">' +
                            '<div class="server-status ' + (isHealthy ? 'healthy' : 'unhealthy') + '"></div>' +
                            '<div class="server-info">' +
                                '<div class="server-title">' + server.Name + '</div>' +
                                '<div class="server-endpoint">Weight: ' + server.Weight + '</div>' +
                            '</div>' +
                        '</div>' +
                    '</td>' +
                    '<td><span class="badge ' + providerClass + '">' + providerName + '</span></td>' +
                    '<td class="mono text-muted">' + server.Hostgroup + '</td>' +
                    '<td class="text-right mono">' + server.TotalReqs.toLocaleString() + '</td>' +
                    '<td class="text-right mono">' + latencyDisplay + '</td>' +
                    '<td class="text-right"><span class="badge ' + (isHealthy ? 'badge-healthy' : 'badge-unhealthy') + '">' + (isHealthy ? 'Healthy' : 'Unhealthy') + '</span></td>' +
                '</tr>';
            }).join('');
        }
        
        function renderHealth(healthData) {
            var container = document.getElementById('healthList');
            var entries = Object.entries(healthData);
            
            if (entries.length === 0) {
                container.innerHTML = '<div class="empty-state"><div class="empty-icon">💚</div>No health data</div>';
                return;
            }
            
            container.innerHTML = entries.map(function(entry) {
                var name = entry[0];
                var status = entry[1];
                var latencyMs = status.Latency > 0 ? Math.round(status.Latency / 1000000) : 0;
                var latencyDisplay = latencyMs > 0 ? latencyMs + ' ms' : '- ms';
                
                return '<div class="health-item">' +
                    '<div class="health-name">' +
                        '<div class="server-status ' + (status.Healthy ? 'healthy' : 'unhealthy') + '"></div>' +
                        '<span>' + name + '</span>' +
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
            var status = document.getElementById('connectionStatus');
            status.innerHTML = '<span class="status-dot error"></span><span>Disconnected</span>';
        }
        
        function setOnline() {
            var status = document.getElementById('connectionStatus');
            status.innerHTML = '<span class="status-dot"></span><span>Connected</span>';
        }
        
        function updateLastUpdated() {
            lastUpdate = new Date();
            document.getElementById('lastUpdated').textContent = 'Updated just now';
        }
        
        // Update "time ago" display
        setInterval(function() {
            var seconds = Math.floor((new Date() - lastUpdate) / 1000);
            var text = 'Updated ';
            if (seconds < 5) text += 'just now';
            else if (seconds < 60) text += seconds + 's ago';
            else text += Math.floor(seconds / 60) + 'm ago';
            document.getElementById('lastUpdated').textContent = text;
        }, 1000);
        
        async function refreshAll() {
            await Promise.all([fetchStats(), fetchHealth()]);
        }
        
        // Initial load
        refreshAll();
        
        // Auto-refresh every 5 seconds
        setInterval(refreshAll, 5000);
    </script>
</body>
</html>`
