package admin

// adminPanelHTML contains the embedded admin panel dashboard.
// This requires authentication to access.
const adminPanelHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MindBalancer - Admin</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Google+Sans:wght@400;500;700&family=Roboto:wght@400;500&family=Roboto+Mono:wght@400&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8f9fa;
            --bg-tertiary: #f1f3f4;
            --bg-hover: #e8eaed;
            --bg-active: #e3f2fd;
            --border: #dadce0;
            --border-light: #e0e0e0;
            --text-primary: #202124;
            --text-secondary: #5f6368;
            --text-muted: #80868b;
            --accent-blue: #1a73e8;
            --accent-blue-hover: #1967d2;
            --accent-green: #1e8e3e;
            --accent-green-bg: #e6f4ea;
            --accent-red: #d93025;
            --accent-red-bg: #fce8e6;
            --accent-orange: #e37400;
            --accent-orange-bg: #fef7e0;
            --sidebar-width: 256px;
            --header-height: 64px;
            --shadow-1: 0 1px 2px 0 rgba(60,64,67,0.3), 0 1px 3px 1px rgba(60,64,67,0.15);
            --shadow-2: 0 1px 2px 0 rgba(60,64,67,0.3), 0 2px 6px 2px rgba(60,64,67,0.15);
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Google Sans', 'Roboto', sans-serif;
            background: var(--bg-secondary);
            color: var(--text-primary);
            min-height: 100vh;
        }

        /* Header */
        .header {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: var(--header-height);
            background: var(--bg-primary);
            border-bottom: 1px solid var(--border);
            display: flex;
            align-items: center;
            padding: 0 16px;
            z-index: 100;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
            width: var(--sidebar-width);
            padding-right: 16px;
        }

        .logo-icon {
            width: 40px;
            height: 40px;
            border-radius: 8px;
            object-fit: contain;
        }

        .logo-text {
            font-size: 20px;
            font-weight: 500;
            color: var(--text-primary);
        }

        .header-content {
            flex: 1;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .header-title {
            font-size: 18px;
            font-weight: 500;
            color: var(--text-primary);
        }

        .header-actions {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        /* Sidebar */
        .sidebar {
            position: fixed;
            top: var(--header-height);
            left: 0;
            bottom: 0;
            width: var(--sidebar-width);
            background: var(--bg-primary);
            border-right: 1px solid var(--border);
            overflow-y: auto;
            padding: 8px 0;
        }

        .nav-section {
            padding: 8px 0;
        }

        .nav-section-title {
            padding: 8px 24px;
            font-size: 11px;
            font-weight: 500;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .nav-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 0 24px;
            height: 40px;
            color: var(--text-secondary);
            cursor: pointer;
            transition: all 0.15s ease;
            text-decoration: none;
            font-size: 14px;
            font-weight: 500;
        }

        .nav-item:hover {
            background: var(--bg-hover);
            color: var(--text-primary);
        }

        .nav-item.active {
            background: var(--bg-active);
            color: var(--accent-blue);
        }

        .nav-item.active .nav-icon {
            color: var(--accent-blue);
        }

        .nav-icon {
            font-size: 20px;
            width: 24px;
            text-align: center;
        }

        .nav-divider {
            height: 1px;
            background: var(--border);
            margin: 8px 0;
        }

        /* Main Content */
        .main {
            margin-left: var(--sidebar-width);
            margin-top: var(--header-height);
            padding: 24px;
            min-height: calc(100vh - var(--header-height));
        }

        /* Page Container */
        .page {
            display: none;
        }

        .page.active {
            display: block;
        }

        /* Page Header */
        .page-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 24px;
        }

        .page-title {
            font-size: 24px;
            font-weight: 400;
            color: var(--text-primary);
        }

        /* Buttons */
        .btn {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            padding: 8px 24px;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.15s ease;
            border: none;
            font-family: inherit;
        }

        .btn-primary {
            background: var(--accent-blue);
            color: white;
        }

        .btn-primary:hover {
            background: var(--accent-blue-hover);
            box-shadow: var(--shadow-1);
        }

        .btn-secondary {
            background: var(--bg-primary);
            color: var(--accent-blue);
            border: 1px solid var(--border);
        }

        .btn-secondary:hover {
            background: var(--bg-tertiary);
        }

        .btn-danger {
            background: var(--accent-red);
            color: white;
        }

        .btn-danger:hover {
            background: #c5221f;
        }

        .btn-text {
            background: transparent;
            color: var(--accent-blue);
            padding: 8px 12px;
        }

        .btn-text:hover {
            background: var(--bg-hover);
        }

        .btn-icon {
            padding: 8px;
            border-radius: 50%;
            background: transparent;
            border: none;
            color: var(--text-secondary);
            cursor: pointer;
        }

        .btn-icon:hover {
            background: var(--bg-hover);
            color: var(--text-primary);
        }

        /* Cards */
        .card {
            background: var(--bg-primary);
            border-radius: 8px;
            border: 1px solid var(--border);
            margin-bottom: 24px;
        }

        .card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 24px;
            border-bottom: 1px solid var(--border);
        }

        .card-title {
            font-size: 16px;
            font-weight: 500;
        }

        .card-body {
            padding: 24px;
        }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
            margin-bottom: 24px;
        }

        .stat-card {
            background: var(--bg-primary);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 20px;
        }

        .stat-label {
            font-size: 12px;
            color: var(--text-muted);
            margin-bottom: 8px;
        }

        .stat-value {
            font-size: 28px;
            font-weight: 400;
            color: var(--text-primary);
            font-family: 'Roboto', sans-serif;
        }

        .stat-value.green { color: var(--accent-green); }
        .stat-value.red { color: var(--accent-red); }
        .stat-value.blue { color: var(--accent-blue); }

        /* Table */
        .table-container {
            overflow-x: auto;
        }

        .table {
            width: 100%;
            border-collapse: collapse;
        }

        .table th {
            text-align: left;
            padding: 12px 16px;
            font-size: 12px;
            font-weight: 500;
            color: var(--text-secondary);
            border-bottom: 1px solid var(--border);
            background: var(--bg-secondary);
        }

        .table td {
            padding: 16px;
            font-size: 14px;
            border-bottom: 1px solid var(--border);
            vertical-align: middle;
        }

        .table tr:hover {
            background: var(--bg-tertiary);
        }

        .table tr:last-child td {
            border-bottom: none;
        }

        /* Status Badges */
        .status {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 500;
        }

        .status-online, .status-healthy, .status-active {
            background: var(--accent-green-bg);
            color: var(--accent-green);
        }

        .status-offline, .status-unhealthy, .status-inactive {
            background: var(--accent-red-bg);
            color: var(--accent-red);
        }

        .status-warning {
            background: var(--accent-orange-bg);
            color: var(--accent-orange);
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: currentColor;
        }

        /* Provider Badge */
        .provider-badge {
            display: inline-flex;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 500;
            background: var(--bg-tertiary);
            color: var(--text-secondary);
        }

        /* Actions */
        .actions {
            display: flex;
            gap: 4px;
        }

        /* Modal */
        .modal-overlay {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.5);
            z-index: 200;
            align-items: center;
            justify-content: center;
        }

        .modal-overlay.active {
            display: flex;
        }

        .modal {
            background: var(--bg-primary);
            border-radius: 8px;
            width: 100%;
            max-width: 560px;
            max-height: 90vh;
            overflow: hidden;
            box-shadow: var(--shadow-2);
        }

        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 24px;
            border-bottom: 1px solid var(--border);
        }

        .modal-title {
            font-size: 20px;
            font-weight: 500;
        }

        .modal-body {
            padding: 24px;
            overflow-y: auto;
            max-height: calc(90vh - 140px);
        }

        .modal-footer {
            display: flex;
            justify-content: flex-end;
            gap: 8px;
            padding: 16px 24px;
            border-top: 1px solid var(--border);
        }

        /* Form */
        .form-group {
            margin-bottom: 20px;
        }

        .form-label {
            display: block;
            font-size: 14px;
            font-weight: 500;
            color: var(--text-primary);
            margin-bottom: 8px;
        }

        .form-input {
            width: 100%;
            padding: 12px 16px;
            border: 1px solid var(--border);
            border-radius: 4px;
            font-size: 14px;
            font-family: inherit;
            transition: border-color 0.15s ease;
        }

        .form-input:focus {
            outline: none;
            border-color: var(--accent-blue);
        }

        .form-select {
            width: 100%;
            padding: 12px 16px;
            border: 1px solid var(--border);
            border-radius: 4px;
            font-size: 14px;
            font-family: inherit;
            background: var(--bg-primary);
            cursor: pointer;
        }

        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 16px;
        }

        .form-hint {
            font-size: 12px;
            color: var(--text-muted);
            margin-top: 4px;
        }

        /* Toggle */
        .toggle {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .toggle-switch {
            position: relative;
            width: 48px;
            height: 24px;
            background: var(--border);
            border-radius: 12px;
            cursor: pointer;
            transition: background 0.2s ease;
        }

        .toggle-switch.active {
            background: var(--accent-blue);
        }

        .toggle-switch::after {
            content: '';
            position: absolute;
            top: 2px;
            left: 2px;
            width: 20px;
            height: 20px;
            background: white;
            border-radius: 50%;
            transition: transform 0.2s ease;
            box-shadow: 0 1px 3px rgba(0,0,0,0.3);
        }

        .toggle-switch.active::after {
            transform: translateX(24px);
        }

        /* Mono */
        .mono {
            font-family: 'Roboto Mono', monospace;
        }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 48px;
            color: var(--text-muted);
        }

        .empty-state-icon {
            font-size: 48px;
            margin-bottom: 16px;
        }

        /* Alert */
        .alert {
            padding: 16px 20px;
            border-radius: 4px;
            margin-bottom: 16px;
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .alert-success {
            background: var(--accent-green-bg);
            color: var(--accent-green);
        }

        .alert-error {
            background: var(--accent-red-bg);
            color: var(--accent-red);
        }

        .alert-info {
            background: #e8f0fe;
            color: var(--accent-blue);
        }

        /* Toast */
        .toast-container {
            position: fixed;
            bottom: 24px;
            left: 50%;
            transform: translateX(-50%);
            z-index: 300;
        }

        .toast {
            background: #323232;
            color: white;
            padding: 16px 24px;
            border-radius: 4px;
            box-shadow: var(--shadow-2);
            animation: slideUp 0.3s ease;
        }

        @keyframes slideUp {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }

        /* Responsive */
        @media (max-width: 1200px) {
            .stats-grid { grid-template-columns: repeat(2, 1fr); }
        }

        @media (max-width: 768px) {
            .sidebar { display: none; }
            .main { margin-left: 0; }
            .stats-grid { grid-template-columns: 1fr; }
            .form-row { grid-template-columns: 1fr; }
        }
    </style>
</head>
<body>
    <!-- Header -->
    <header class="header">
        <div class="logo">
            <img src="/static/logo-mindbalancer.png" alt="MindBalancer" class="logo-icon">
            <span class="logo-text">MindBalancer</span>
        </div>
        <div class="header-content">
            <span class="header-title" id="pageTitle">Overview</span>
            <div class="header-actions">
                <a href="/monitoring" class="btn btn-secondary">View Monitoring</a>
                <a href="/admin/logout" class="btn btn-text">Logout</a>
            </div>
        </div>
    </header>

    <!-- Sidebar -->
    <nav class="sidebar">
        <div class="nav-section">
            <a class="nav-item active" data-page="overview" onclick="showPage('overview')">
                <svg class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
                Overview
            </a>
        </div>
        <div class="nav-divider"></div>
        <div class="nav-section">
            <div class="nav-section-title">Resources</div>
            <a class="nav-item" data-page="servers" onclick="showPage('servers')">
                <svg class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><circle cx="6" cy="6" r="1" fill="currentColor"/><circle cx="6" cy="18" r="1" fill="currentColor"/></svg>
                Servers
            </a>
            <a class="nav-item" data-page="users" onclick="showPage('users')">
                <svg class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
                Users
            </a>
            <a class="nav-item" data-page="rules" onclick="showPage('rules')">
                <svg class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 3 21 3 21 8"/><line x1="4" y1="20" x2="21" y2="3"/><polyline points="21 16 21 21 16 21"/><line x1="15" y1="15" x2="21" y2="21"/><line x1="4" y1="4" x2="9" y2="9"/></svg>
                Routing Rules
            </a>
            <a class="nav-item" data-page="hostgroups" onclick="showPage('hostgroups')">
                <svg class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                Hostgroups
            </a>
        </div>
        <div class="nav-divider"></div>
        <div class="nav-section">
            <div class="nav-section-title">Configuration</div>
            <a class="nav-item" data-page="variables" onclick="showPage('variables')">
                <svg class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
                Variables
            </a>
            <a class="nav-item" data-page="cache" onclick="showPage('cache')">
                <svg class="nav-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>
                Cache
            </a>
        </div>
    </nav>

    <!-- Main Content -->
    <main class="main">
        <!-- Toast Container -->
        <div class="toast-container" id="toastContainer"></div>

        <!-- Overview Page -->
        <div class="page active" id="page-overview">
            <div class="page-header">
                <h1 class="page-title">Overview</h1>
                <div style="display:flex;gap:8px;">
                    <button class="btn btn-secondary" onclick="reloadConfig()">
                        Reload Configuration
                    </button>
                </div>
            </div>

            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">Total Servers</div>
                    <div class="stat-value" id="overview-totalServers">-</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Healthy Servers</div>
                    <div class="stat-value green" id="overview-healthyServers">-</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Total Requests</div>
                    <div class="stat-value blue" id="overview-totalRequests">-</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Cache Hit Rate</div>
                    <div class="stat-value" id="overview-cacheHitRate">-</div>
                </div>
            </div>

            <div style="display:grid;grid-template-columns:1fr 1fr;gap:24px;">
                <div class="card">
                    <div class="card-header">
                        <span class="card-title">System Status</span>
                    </div>
                    <div class="card-body">
                        <table class="table">
                            <tbody>
                                <tr><td>Algorithm</td><td class="mono" id="overview-algorithm">-</td></tr>
                                <tr><td>Total Users</td><td class="mono" id="overview-totalUsers">-</td></tr>
                                <tr><td>Routing Rules</td><td class="mono" id="overview-totalRules">-</td></tr>
                                <tr><td>Hostgroups</td><td class="mono" id="overview-totalHostgroups">-</td></tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <div class="card">
                    <div class="card-header">
                        <span class="card-title">Quick Actions</span>
                    </div>
                    <div class="card-body">
                        <div style="display:flex;flex-direction:column;gap:12px;">
                            <button class="btn btn-secondary" onclick="showPage('servers')">
                                Manage Servers
                            </button>
                            <button class="btn btn-secondary" onclick="showPage('cache')">
                                Cache Management
                            </button>
                            <button class="btn btn-secondary" onclick="showPage('variables')">
                                Edit Variables
                            </button>
                            <button class="btn btn-secondary" onclick="clearCache()">
                                Clear Cache
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            <div class="card" style="margin-top:24px;">
                <div class="card-header">
                    <span class="card-title">Server Health</span>
                </div>
                <div class="card-body" id="overview-health">
                    Loading...
                </div>
            </div>
        </div>

        <!-- Servers Page -->
        <div class="page" id="page-servers">
            <div class="page-header">
                <h1 class="page-title">Servers</h1>
                <button class="btn btn-primary" onclick="openServerModal()">
                    <span>+</span> Add Server
                </button>
            </div>

            <div class="card">
                <div class="table-container">
                    <table class="table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Provider</th>
                                <th>Endpoint</th>
                                <th>Hostgroup</th>
                                <th>Weight</th>
                                <th>Status</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="servers-table">
                            <tr><td colspan="7" class="empty-state">Loading...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- Users Page -->
        <div class="page" id="page-users">
            <div class="page-header">
                <h1 class="page-title">Users</h1>
                <button class="btn btn-primary" onclick="openUserModal()">
                    <span>+</span> Add User
                </button>
            </div>

            <div class="card">
                <div class="table-container">
                    <table class="table">
                        <thead>
                            <tr>
                                <th>Username</th>
                                <th>Status</th>
                                <th>Requests/min</th>
                                <th>Tokens/min</th>
                                <th>Default Hostgroup</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="users-table">
                            <tr><td colspan="6" class="empty-state">Loading...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- Rules Page -->
        <div class="page" id="page-rules">
            <div class="page-header">
                <h1 class="page-title">Routing Rules</h1>
                <button class="btn btn-primary" onclick="openRuleModal()">
                    <span>+</span> Add Rule
                </button>
            </div>

            <div class="card">
                <div class="table-container">
                    <table class="table">
                        <thead>
                            <tr>
                                <th>Rule ID</th>
                                <th>Match Model</th>
                                <th>Match Pattern</th>
                                <th>Destination HG</th>
                                <th>Priority</th>
                                <th>Status</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="rules-table">
                            <tr><td colspan="7" class="empty-state">Loading...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- Hostgroups Page -->
        <div class="page" id="page-hostgroups">
            <div class="page-header">
                <h1 class="page-title">Hostgroups</h1>
                <button class="btn btn-primary" onclick="openHostgroupModal()">
                    <span>+</span> Add Hostgroup
                </button>
            </div>

            <div class="card">
                <div class="table-container">
                    <table class="table">
                        <thead>
                            <tr>
                                <th>Group ID</th>
                                <th>Name</th>
                                <th>Comment</th>
                                <th>Server Count</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="hostgroups-table">
                            <tr><td colspan="5" class="empty-state">Loading...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- Variables Page -->
        <div class="page" id="page-variables">
            <div class="page-header">
                <h1 class="page-title">Variables</h1>
                <button class="btn btn-primary" onclick="reloadConfig()">
                    Reload Configuration
                </button>
            </div>

            <div class="alert alert-info" id="variables-reload-hint" style="display:none;">
                Variables updated. Click "Reload Configuration" to apply changes to runtime.
            </div>

            <div class="card">
                <div class="card-header">
                    <span class="card-title">Runtime Variables</span>
                    <span class="card-badge">Editable</span>
                </div>
                <div class="table-container">
                    <table class="table">
                        <thead>
                            <tr>
                                <th>Variable Name</th>
                                <th>Value</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody id="variables-table">
                            <tr><td colspan="3" class="empty-state">Loading...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- Cache Page -->
        <div class="page" id="page-cache">
            <div class="page-header">
                <h1 class="page-title">Cache Management</h1>
            </div>

            <div class="stats-grid">
                <div class="stat-card">
                    <div class="stat-label">Hit Rate</div>
                    <div class="stat-value blue" id="cache-hitRate">-</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Total Hits</div>
                    <div class="stat-value" id="cache-hits">-</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Memory Used</div>
                    <div class="stat-value" id="cache-memory">-</div>
                </div>
                <div class="stat-card">
                    <div class="stat-label">Item Count</div>
                    <div class="stat-value" id="cache-items">-</div>
                </div>
            </div>

            <div class="card">
                <div class="card-header">
                    <span class="card-title">Cache Control</span>
                </div>
                <div class="card-body">
                    <div class="toggle" style="margin-bottom: 24px;">
                        <div class="toggle-switch" id="cache-toggle" onclick="toggleCache()"></div>
                        <span>Cache <span id="cache-status-text">Disabled</span></span>
                    </div>
                    <button class="btn btn-danger" onclick="clearCache()">
                        Clear Cache
                    </button>
                </div>
            </div>

            <div class="card">
                <div class="card-header">
                    <span class="card-title">Cache Statistics</span>
                </div>
                <div class="card-body">
                    <table class="table">
                        <tbody id="cache-stats-table">
                            <tr><td colspan="2" class="empty-state">Loading...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </main>

    <!-- Server Modal -->
    <div class="modal-overlay" id="server-modal">
        <div class="modal">
            <div class="modal-header">
                <span class="modal-title" id="server-modal-title">Add Server</span>
                <button class="btn-icon" onclick="closeModal('server-modal')">&times;</button>
            </div>
            <div class="modal-body">
                <form id="server-form">
                    <input type="hidden" id="server-edit-name">
                    <div class="form-group">
                        <label class="form-label">Name *</label>
                        <input type="text" class="form-input" id="server-name" required placeholder="e.g., openai-primary">
                    </div>
                    <div class="form-group">
                        <label class="form-label">Provider Type *</label>
                        <select class="form-select" id="server-provider">
                            <option value="openai">OpenAI</option>
                            <option value="anthropic">Anthropic</option>
                            <option value="google">Google</option>
                            <option value="azure">Azure OpenAI</option>
                            <option value="groq">Groq</option>
                            <option value="ollama">Ollama</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Endpoint URL *</label>
                        <input type="url" class="form-input" id="server-endpoint" required placeholder="https://api.openai.com/v1">
                    </div>
                    <div class="form-group">
                        <label class="form-label">API Key</label>
                        <input type="password" class="form-input" id="server-apikey" placeholder="sk-...">
                        <div class="form-hint">Leave empty to keep existing key when editing</div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">Hostgroup</label>
                            <input type="number" class="form-input" id="server-hostgroup" value="0" min="0">
                        </div>
                        <div class="form-group">
                            <label class="form-label">Weight</label>
                            <input type="number" class="form-input" id="server-weight" value="1" min="1">
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">Max Connections</label>
                            <input type="number" class="form-input" id="server-maxconn" value="100" min="1">
                        </div>
                        <div class="form-group">
                            <label class="form-label">Status</label>
                            <select class="form-select" id="server-status">
                                <option value="ONLINE">Online</option>
                                <option value="OFFLINE">Offline</option>
                            </select>
                        </div>
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" onclick="closeModal('server-modal')">Cancel</button>
                <button class="btn btn-primary" onclick="saveServer()">Save</button>
            </div>
        </div>
    </div>

    <!-- User Modal -->
    <div class="modal-overlay" id="user-modal">
        <div class="modal">
            <div class="modal-header">
                <span class="modal-title" id="user-modal-title">Add User</span>
                <button class="btn-icon" onclick="closeModal('user-modal')">&times;</button>
            </div>
            <div class="modal-body">
                <form id="user-form">
                    <input type="hidden" id="user-edit-name">
                    <div class="form-group">
                        <label class="form-label">Username *</label>
                        <input type="text" class="form-input" id="user-name" required placeholder="e.g., api-user-1">
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">Requests per Minute</label>
                            <input type="number" class="form-input" id="user-rpm" value="1000" min="1">
                        </div>
                        <div class="form-group">
                            <label class="form-label">Tokens per Minute</label>
                            <input type="number" class="form-input" id="user-tpm" value="100000" min="1">
                        </div>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Default Hostgroup</label>
                        <input type="number" class="form-input" id="user-hostgroup" value="0" min="0">
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" onclick="closeModal('user-modal')">Cancel</button>
                <button class="btn btn-primary" onclick="saveUser()">Save</button>
            </div>
        </div>
    </div>

    <!-- Rule Modal -->
    <div class="modal-overlay" id="rule-modal">
        <div class="modal">
            <div class="modal-header">
                <span class="modal-title" id="rule-modal-title">Add Routing Rule</span>
                <button class="btn-icon" onclick="closeModal('rule-modal')">&times;</button>
            </div>
            <div class="modal-body">
                <form id="rule-form">
                    <input type="hidden" id="rule-edit-id">
                    <div class="form-group">
                        <label class="form-label">Match Model</label>
                        <input type="text" class="form-input" id="rule-model" placeholder="e.g., gpt-4, claude-*">
                        <div class="form-hint">Leave empty to match all models. Supports wildcards (*)</div>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Match Pattern</label>
                        <input type="text" class="form-input" id="rule-pattern" placeholder="e.g., /v1/chat/completions">
                        <div class="form-hint">Request path pattern to match</div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">Destination Hostgroup *</label>
                            <input type="number" class="form-input" id="rule-hostgroup" value="0" min="0" required>
                        </div>
                        <div class="form-group">
                            <label class="form-label">Priority</label>
                            <input type="number" class="form-input" id="rule-priority" value="100" min="1">
                            <div class="form-hint">Higher = matched first</div>
                        </div>
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" onclick="closeModal('rule-modal')">Cancel</button>
                <button class="btn btn-primary" onclick="saveRule()">Save</button>
            </div>
        </div>
    </div>

    <!-- Hostgroup Modal -->
    <div class="modal-overlay" id="hostgroup-modal">
        <div class="modal">
            <div class="modal-header">
                <span class="modal-title">Add Hostgroup</span>
                <button class="btn-icon" onclick="closeModal('hostgroup-modal')">&times;</button>
            </div>
            <div class="modal-body">
                <form id="hostgroup-form">
                    <div class="form-group">
                        <label class="form-label">Group ID *</label>
                        <input type="number" class="form-input" id="hostgroup-id" value="0" min="0" required>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Name *</label>
                        <input type="text" class="form-input" id="hostgroup-name" required placeholder="e.g., primary-openai">
                    </div>
                    <div class="form-group">
                        <label class="form-label">Comment</label>
                        <input type="text" class="form-input" id="hostgroup-comment" placeholder="Optional description">
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" onclick="closeModal('hostgroup-modal')">Cancel</button>
                <button class="btn btn-primary" onclick="saveHostgroup()">Save</button>
            </div>
        </div>
    </div>

    <!-- Variable Modal -->
    <div class="modal-overlay" id="variable-modal">
        <div class="modal">
            <div class="modal-header">
                <span class="modal-title">Edit Variable</span>
                <button class="btn-icon" onclick="closeModal('variable-modal')">&times;</button>
            </div>
            <div class="modal-body">
                <form id="variable-form">
                    <div class="form-group">
                        <label class="form-label">Variable Name</label>
                        <input type="text" class="form-input" id="variable-name" readonly>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Value</label>
                        <input type="text" class="form-input" id="variable-value">
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" onclick="closeModal('variable-modal')">Cancel</button>
                <button class="btn btn-primary" onclick="saveVariable()">Save</button>
            </div>
        </div>
    </div>

    <script>
        // Page Navigation
        function showPage(page) {
            document.querySelectorAll('.page').forEach(function(p) { p.classList.remove('active'); });
            document.querySelectorAll('.nav-item').forEach(function(n) { n.classList.remove('active'); });
            
            document.getElementById('page-' + page).classList.add('active');
            document.querySelector('[data-page="' + page + '"]').classList.add('active');
            
            var titles = {
                'overview': 'Overview',
                'servers': 'Servers',
                'users': 'Users',
                'rules': 'Routing Rules',
                'hostgroups': 'Hostgroups',
                'variables': 'Variables',
                'cache': 'Cache'
            };
            document.getElementById('pageTitle').textContent = titles[page] || page;
            
            // Load data for the page
            switch(page) {
                case 'overview': loadOverview(); break;
                case 'servers': loadServers(); break;
                case 'users': loadUsers(); break;
                case 'rules': loadRules(); break;
                case 'hostgroups': loadHostgroups(); break;
                case 'variables': loadVariables(); break;
                case 'cache': loadCache(); break;
            }
        }

        // Modal Functions
        function openModal(id) {
            document.getElementById(id).classList.add('active');
        }

        function closeModal(id) {
            document.getElementById(id).classList.remove('active');
        }

        // Toast
        function showToast(message, type) {
            var container = document.getElementById('toastContainer');
            var toast = document.createElement('div');
            toast.className = 'toast';
            toast.textContent = message;
            container.appendChild(toast);
            setTimeout(function() { toast.remove(); }, 3000);
        }

        // API Helpers
        async function apiGet(endpoint) {
            var response = await fetch('/api' + endpoint);
            if (!response.ok) throw new Error('API error');
            return response.json();
        }

        async function apiPost(endpoint, data) {
            var response = await fetch('/api' + endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            if (!response.ok) {
                var err = await response.json();
                throw new Error(err.error || 'API error');
            }
            return response.json();
        }

        async function apiPut(endpoint, data) {
            var response = await fetch('/api' + endpoint, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            if (!response.ok) {
                var err = await response.json();
                throw new Error(err.error || 'API error');
            }
            return response.json();
        }

        async function apiDelete(endpoint) {
            var response = await fetch('/api' + endpoint, { method: 'DELETE' });
            if (!response.ok) throw new Error('API error');
            return true;
        }

        // Overview
        async function loadOverview() {
            try {
                var [stats, users, rules, hostgroups, cache, health] = await Promise.all([
                    apiGet('/stats'),
                    apiGet('/users'),
                    apiGet('/rules'),
                    apiGet('/hostgroups'),
                    fetch('/api/cache').then(r => r.json()).catch(() => ({})),
                    fetch('/api/health').then(r => r.json()).catch(() => ({}))
                ]);
                
                // Stats cards
                document.getElementById('overview-totalServers').textContent = stats.TotalServers || 0;
                document.getElementById('overview-healthyServers').textContent = stats.HealthyServers || 0;
                document.getElementById('overview-totalRequests').textContent = formatNumber(getTotalRequests(stats));
                
                var hitRate = cache.hit_rate !== undefined ? (cache.hit_rate * 100).toFixed(1) + '%' : '-';
                document.getElementById('overview-cacheHitRate').textContent = hitRate;
                
                // System status table
                document.getElementById('overview-algorithm').textContent = stats.Algorithm || '-';
                document.getElementById('overview-totalUsers').textContent = users.length || 0;
                document.getElementById('overview-totalRules').textContent = rules.length || 0;
                document.getElementById('overview-totalHostgroups').textContent = hostgroups.length || 0;
                
                // Health grid
                var healthEntries = Object.entries(health);
                if (healthEntries.length > 0) {
                    var healthHtml = '<div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:12px;">';
                    healthEntries.forEach(function(entry) {
                        var name = entry[0];
                        var status = entry[1];
                        var statusClass = status.Healthy ? 'status-healthy' : 'status-unhealthy';
                        var latencyMs = status.Latency > 0 ? Math.round(status.Latency / 1000000) + 'ms' : '-';
                        healthHtml += '<div style="display:flex;justify-content:space-between;align-items:center;padding:12px;background:#f8f9fa;border-radius:6px;border:1px solid #e0e0e0;">';
                        healthHtml += '<div><span class="status ' + statusClass + '" style="margin-right:8px;"><span class="status-dot"></span>' + (status.Healthy ? 'OK' : 'Down') + '</span><strong>' + name + '</strong></div>';
                        healthHtml += '<span class="mono" style="color:#666;">' + latencyMs + '</span>';
                        healthHtml += '</div>';
                    });
                    healthHtml += '</div>';
                    document.getElementById('overview-health').innerHTML = healthHtml;
                } else {
                    document.getElementById('overview-health').innerHTML = '<div style="color:#888;">No servers configured</div>';
                }
            } catch (error) {
                console.error('Failed to load overview:', error);
            }
        }

        function getTotalRequests(stats) {
            var total = 0;
            if (stats.ServerStats) {
                stats.ServerStats.forEach(function(s) { total += s.TotalReqs || 0; });
            }
            return total;
        }

        async function reloadConfig() {
            try {
                await apiPost('/reload', {});
                showToast('Configuration reloaded successfully', 'success');
                loadOverview();
            } catch (error) {
                showToast('Failed to reload: ' + error.message, 'error');
            }
        }

        // Servers
        async function loadServers() {
            try {
                var servers = await apiGet('/servers');
                var tbody = document.getElementById('servers-table');
                
                if (!servers || servers.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No servers configured</td></tr>';
                    return;
                }
                
                tbody.innerHTML = servers.map(function(s) {
                    var statusClass = s.healthy ? 'status-online' : 'status-offline';
                    var statusText = s.healthy ? 'Healthy' : 'Unhealthy';
                    
                    return '<tr>' +
                        '<td><strong>' + s.name + '</strong></td>' +
                        '<td><span class="provider-badge">' + s.provider_type + '</span></td>' +
                        '<td class="mono" style="max-width:200px;overflow:hidden;text-overflow:ellipsis;">' + s.endpoint + '</td>' +
                        '<td>' + s.hostgroup + '</td>' +
                        '<td>' + s.weight + '</td>' +
                        '<td><span class="status ' + statusClass + '"><span class="status-dot"></span>' + statusText + '</span></td>' +
                        '<td class="actions">' +
                            '<button class="btn-icon" onclick="editServer(\'' + s.name + '\')" title="Edit"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></button>' +
                            '<button class="btn-icon" onclick="deleteServer(\'' + s.name + '\')" title="Delete"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>' +
                        '</td>' +
                    '</tr>';
                }).join('');
            } catch (error) {
                console.error('Failed to load servers:', error);
            }
        }

        function openServerModal(server) {
            document.getElementById('server-modal-title').textContent = server ? 'Edit Server' : 'Add Server';
            document.getElementById('server-edit-name').value = server ? server.name : '';
            document.getElementById('server-name').value = server ? server.name : '';
            document.getElementById('server-name').disabled = !!server;
            document.getElementById('server-provider').value = server ? server.provider_type : 'openai';
            document.getElementById('server-endpoint').value = server ? server.endpoint : '';
            document.getElementById('server-apikey').value = '';
            document.getElementById('server-hostgroup').value = server ? server.hostgroup : 0;
            document.getElementById('server-weight').value = server ? server.weight : 1;
            document.getElementById('server-maxconn').value = server ? server.max_connections : 100;
            document.getElementById('server-status').value = server ? server.status : 'ONLINE';
            openModal('server-modal');
        }

        async function editServer(name) {
            try {
                var server = await apiGet('/servers/' + name);
                openServerModal(server);
            } catch (error) {
                showToast('Failed to load server', 'error');
            }
        }

        async function saveServer() {
            var editName = document.getElementById('server-edit-name').value;
            var data = {
                name: document.getElementById('server-name').value,
                provider_type: document.getElementById('server-provider').value,
                endpoint: document.getElementById('server-endpoint').value,
                hostgroup: parseInt(document.getElementById('server-hostgroup').value),
                weight: parseInt(document.getElementById('server-weight').value),
                max_connections: parseInt(document.getElementById('server-maxconn').value),
                status: document.getElementById('server-status').value
            };
            
            var apiKey = document.getElementById('server-apikey').value;
            if (apiKey) {
                data.api_key_encrypted = apiKey;
            }
            
            try {
                if (editName) {
                    await apiPut('/servers/' + editName, data);
                    showToast('Server updated', 'success');
                } else {
                    await apiPost('/servers', data);
                    showToast('Server created', 'success');
                }
                closeModal('server-modal');
                loadServers();
            } catch (error) {
                showToast('Error: ' + error.message, 'error');
            }
        }

        async function deleteServer(name) {
            if (!confirm('Delete server "' + name + '"?')) return;
            try {
                await apiDelete('/servers/' + name);
                showToast('Server deleted', 'success');
                loadServers();
            } catch (error) {
                showToast('Failed to delete server', 'error');
            }
        }

        // Users
        async function loadUsers() {
            try {
                var users = await apiGet('/users');
                var tbody = document.getElementById('users-table');
                
                if (!users || users.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="6" class="empty-state">No users configured</td></tr>';
                    return;
                }
                
                tbody.innerHTML = users.map(function(u) {
                    var statusClass = u.active ? 'status-active' : 'status-inactive';
                    var statusText = u.active ? 'Active' : 'Inactive';
                    
                    return '<tr>' +
                        '<td><strong>' + u.username + '</strong></td>' +
                        '<td><span class="status ' + statusClass + '">' + statusText + '</span></td>' +
                        '<td class="mono">' + formatNumber(u.max_requests_per_minute) + '</td>' +
                        '<td class="mono">' + formatNumber(u.max_tokens_per_minute) + '</td>' +
                        '<td>' + (u.default_hostgroup || 0) + '</td>' +
                        '<td class="actions">' +
                            '<button class="btn-icon" onclick="editUser(\'' + u.username + '\')" title="Edit"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></button>' +
                            '<button class="btn-icon" onclick="deleteUser(\'' + u.username + '\')" title="Delete"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>' +
                        '</td>' +
                    '</tr>';
                }).join('');
            } catch (error) {
                console.error('Failed to load users:', error);
            }
        }

        function openUserModal(user) {
            document.getElementById('user-modal-title').textContent = user ? 'Edit User' : 'Add User';
            document.getElementById('user-edit-name').value = user ? user.username : '';
            document.getElementById('user-name').value = user ? user.username : '';
            document.getElementById('user-name').disabled = !!user;
            document.getElementById('user-rpm').value = user ? user.max_requests_per_minute : 1000;
            document.getElementById('user-tpm').value = user ? user.max_tokens_per_minute : 100000;
            document.getElementById('user-hostgroup').value = user ? (user.default_hostgroup || 0) : 0;
            openModal('user-modal');
        }

        async function editUser(username) {
            try {
                var user = await apiGet('/users/' + username);
                openUserModal(user);
            } catch (error) {
                showToast('Failed to load user', 'error');
            }
        }

        async function saveUser() {
            var editName = document.getElementById('user-edit-name').value;
            var data = {
                username: document.getElementById('user-name').value,
                max_requests_per_minute: parseInt(document.getElementById('user-rpm').value),
                max_tokens_per_minute: parseInt(document.getElementById('user-tpm').value),
                default_hostgroup: parseInt(document.getElementById('user-hostgroup').value),
                active: true
            };
            
            try {
                if (editName) {
                    await apiPut('/users/' + editName, data);
                    showToast('User updated', 'success');
                } else {
                    await apiPost('/users', data);
                    showToast('User created', 'success');
                }
                closeModal('user-modal');
                loadUsers();
            } catch (error) {
                showToast('Error: ' + error.message, 'error');
            }
        }

        async function deleteUser(username) {
            if (!confirm('Delete user "' + username + '"?')) return;
            try {
                await apiDelete('/users/' + username);
                showToast('User deleted', 'success');
                loadUsers();
            } catch (error) {
                showToast('Failed to delete user', 'error');
            }
        }

        // Rules
        async function loadRules() {
            try {
                var rules = await apiGet('/rules');
                var tbody = document.getElementById('rules-table');
                
                if (!rules || rules.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No routing rules configured</td></tr>';
                    return;
                }
                
                tbody.innerHTML = rules.map(function(r) {
                    var statusClass = r.active ? 'status-active' : 'status-inactive';
                    var statusText = r.active ? 'Active' : 'Inactive';
                    
                    return '<tr>' +
                        '<td class="mono">' + r.rule_id + '</td>' +
                        '<td>' + (r.match_model || '*') + '</td>' +
                        '<td>' + (r.match_pattern || '*') + '</td>' +
                        '<td>' + r.destination_hostgroup + '</td>' +
                        '<td>' + r.priority + '</td>' +
                        '<td><span class="status ' + statusClass + '">' + statusText + '</span></td>' +
                        '<td class="actions">' +
                            '<button class="btn-icon" onclick="deleteRule(' + r.rule_id + ')" title="Delete"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>' +
                        '</td>' +
                    '</tr>';
                }).join('');
            } catch (error) {
                console.error('Failed to load rules:', error);
            }
        }

        function openRuleModal() {
            document.getElementById('rule-modal-title').textContent = 'Add Routing Rule';
            document.getElementById('rule-edit-id').value = '';
            document.getElementById('rule-model').value = '';
            document.getElementById('rule-pattern').value = '';
            document.getElementById('rule-hostgroup').value = 0;
            document.getElementById('rule-priority').value = 100;
            openModal('rule-modal');
        }

        async function saveRule() {
            var data = {
                match_model: document.getElementById('rule-model').value,
                match_pattern: document.getElementById('rule-pattern').value,
                destination_hostgroup: parseInt(document.getElementById('rule-hostgroup').value),
                priority: parseInt(document.getElementById('rule-priority').value)
            };
            
            try {
                await apiPost('/rules', data);
                showToast('Rule created', 'success');
                closeModal('rule-modal');
                loadRules();
            } catch (error) {
                showToast('Error: ' + error.message, 'error');
            }
        }

        async function deleteRule(ruleId) {
            if (!confirm('Delete rule #' + ruleId + '?')) return;
            try {
                await apiDelete('/rules?rule_id=' + ruleId);
                showToast('Rule deleted', 'success');
                loadRules();
            } catch (error) {
                showToast('Failed to delete rule', 'error');
            }
        }

        // Hostgroups
        async function loadHostgroups() {
            try {
                var hostgroups = await apiGet('/hostgroups');
                var tbody = document.getElementById('hostgroups-table');
                
                if (!hostgroups || hostgroups.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No hostgroups configured</td></tr>';
                    return;
                }
                
                tbody.innerHTML = hostgroups.map(function(h) {
                    return '<tr>' +
                        '<td class="mono">' + h.group_id + '</td>' +
                        '<td><strong>' + h.name + '</strong></td>' +
                        '<td>' + (h.comment || '-') + '</td>' +
                        '<td>' + (h.server_count || 0) + '</td>' +
                        '<td class="actions">' +
                            '<button class="btn-icon" onclick="deleteHostgroup(' + h.group_id + ')" title="Delete"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg></button>' +
                        '</td>' +
                    '</tr>';
                }).join('');
            } catch (error) {
                console.error('Failed to load hostgroups:', error);
            }
        }

        function openHostgroupModal() {
            document.getElementById('hostgroup-id').value = 0;
            document.getElementById('hostgroup-name').value = '';
            document.getElementById('hostgroup-comment').value = '';
            openModal('hostgroup-modal');
        }

        async function saveHostgroup() {
            var data = {
                group_id: parseInt(document.getElementById('hostgroup-id').value),
                name: document.getElementById('hostgroup-name').value,
                comment: document.getElementById('hostgroup-comment').value
            };
            
            try {
                await apiPost('/hostgroups', data);
                showToast('Hostgroup created', 'success');
                closeModal('hostgroup-modal');
                loadHostgroups();
            } catch (error) {
                showToast('Error: ' + error.message, 'error');
            }
        }

        async function deleteHostgroup(groupId) {
            if (!confirm('Delete hostgroup #' + groupId + '?')) return;
            try {
                await apiDelete('/hostgroups?group_id=' + groupId);
                showToast('Hostgroup deleted', 'success');
                loadHostgroups();
            } catch (error) {
                showToast('Failed to delete hostgroup', 'error');
            }
        }

        // Variables
        async function loadVariables() {
            try {
                var vars = await apiGet('/variables');
                var tbody = document.getElementById('variables-table');
                
                var entries = Object.entries(vars);
                if (entries.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="3" class="empty-state">No variables</td></tr>';
                    return;
                }
                
                tbody.innerHTML = entries.map(function(entry) {
                    return '<tr>' +
                        '<td class="mono">' + entry[0] + '</td>' +
                        '<td class="mono">' + entry[1] + '</td>' +
                        '<td class="actions">' +
                            '<button class="btn-icon" onclick="editVariable(\'' + entry[0] + '\', \'' + entry[1] + '\')" title="Edit"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg></button>' +
                        '</td>' +
                    '</tr>';
                }).join('');
            } catch (error) {
                console.error('Failed to load variables:', error);
            }
        }

        function editVariable(name, value) {
            document.getElementById('variable-name').value = name;
            document.getElementById('variable-value').value = value;
            openModal('variable-modal');
        }

        async function saveVariable() {
            var data = {
                name: document.getElementById('variable-name').value,
                value: document.getElementById('variable-value').value
            };
            
            try {
                await apiPut('/variables', data);
                showToast('Variable updated', 'success');
                closeModal('variable-modal');
                loadVariables();
                
                // Show reload hint
                document.getElementById('variables-reload-hint').style.display = 'flex';
                
                // Ask to reload
                if (confirm('Variable updated. Reload configuration now?')) {
                    await reloadConfig();
                    document.getElementById('variables-reload-hint').style.display = 'none';
                }
            } catch (error) {
                showToast('Error: ' + error.message, 'error');
            }
        }

        // Cache
        async function loadCache() {
            try {
                var cache = await apiGet('/cache');
                
                document.getElementById('cache-hitRate').textContent = 
                    cache.hit_rate !== undefined ? (cache.hit_rate * 100).toFixed(1) + '%' : '-';
                document.getElementById('cache-hits').textContent = formatNumber(cache.hits || 0);
                document.getElementById('cache-memory').textContent = formatBytes(cache.memory_used_bytes || 0);
                document.getElementById('cache-items').textContent = formatNumber(cache.item_count || 0);
                
                var toggle = document.getElementById('cache-toggle');
                var statusText = document.getElementById('cache-status-text');
                if (cache.enabled) {
                    toggle.classList.add('active');
                    statusText.textContent = 'Enabled';
                } else {
                    toggle.classList.remove('active');
                    statusText.textContent = 'Disabled';
                }
                
                // Stats table
                var statsHtml = [
                    ['Hits', cache.hits || 0],
                    ['Misses', cache.misses || 0],
                    ['Hit Rate', (cache.hit_rate * 100).toFixed(2) + '%'],
                    ['Evictions', cache.evictions || 0],
                    ['Deduplicated Requests', cache.deduplicated_reqs || 0],
                    ['Compression Saved', formatBytes(cache.compression_saved || 0)],
                    ['Memory Used', formatBytes(cache.memory_used_bytes || 0)],
                    ['Item Count', cache.item_count || 0],
                    ['Avg Item Size', formatBytes(cache.avg_item_size || 0)]
                ].map(function(row) {
                    return '<tr><td>' + row[0] + '</td><td class="mono">' + row[1] + '</td></tr>';
                }).join('');
                
                document.getElementById('cache-stats-table').innerHTML = statsHtml;
            } catch (error) {
                console.error('Failed to load cache:', error);
            }
        }

        async function toggleCache() {
            var toggle = document.getElementById('cache-toggle');
            var newState = !toggle.classList.contains('active');
            
            try {
                await apiPut('/cache', { enabled: newState });
                showToast('Cache ' + (newState ? 'enabled' : 'disabled'), 'success');
                loadCache();
            } catch (error) {
                showToast('Failed to toggle cache', 'error');
            }
        }

        async function clearCache() {
            if (!confirm('Clear all cached data?')) return;
            try {
                await apiPost('/cache/clear', {});
                showToast('Cache cleared', 'success');
                loadCache();
            } catch (error) {
                showToast('Failed to clear cache', 'error');
            }
        }

        // Utility functions
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

        // Initialize
        loadOverview();
    </script>
</body>
</html>`
