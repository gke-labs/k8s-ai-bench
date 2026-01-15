const DATA_URL = "combined_results.jsonl";
const MODEL_KEYWORDS = {
    "gemini": "Proprietary"
};

// Lab/Company identification and styling
const LAB_CONFIG = {
    google: {
        keywords: ["gemini"],
        name: "Google",
        icon: "assets/google.svg"
    },
    qwen: {
        keywords: ["qwen"],
        name: "Qwen",
        icon: "assets/qwen.svg"
    },
    openai: {
        keywords: ["openai", "gpt"],
        name: "OpenAI",
        icon: "assets/openai.svg"
    },
    zhipu: {
        keywords: ["zai", "glm"],
        name: "Zhipu AI",
        icon: "assets/zhipu.svg"
    }
};

// Unique color for each model - visually distinct palette
const MODEL_COLORS = {
    "gemini-2.5-pro": "#4285F4",           // Google Blue
    "gemini-2.5-flash": "#34A853",         // Google Green
    "gemini-3-pro-preview": "#EA4335",     // Google Red
    "Qwen/Qwen3-Coder-480B-A35B-Instruct": "#6366F1",  // Indigo
    "Qwen/Qwen3-Coder-30B-A3B-Instruct": "#8B5CF6",    // Purple
    "Qwen/Qwen3-Next-80B-A3B-Instruct": "#A855F7",     // Violet
    "openai/gpt-oss-120b": "#10A37F",      // OpenAI Teal
    "openai/gpt-oss-20b": "#059669",       // Emerald
    "zai-org/GLM-4.6": "#F97316",          // Orange
    "zai-org/GLM-4.5": "#FB923C"           // Light Orange
};

// Display name overrides for cleaner presentation
const MODEL_DISPLAY_NAMES = {
    "gemini-2.5-pro": "gemini-2.5-pro",
    "gemini-2.5-flash": "gemini-2.5-flash",
    "gemini-3-pro-preview": "gemini-3-pro-preview",
    "Qwen/Qwen3-Coder-480B-A35B-Instruct": "qwen3-coder-480b",
    "Qwen/Qwen3-Coder-30B-A3B-Instruct": "qwen3-coder-30b",
    "Qwen/Qwen3-Next-80B-A3B-Instruct": "qwen3-next-80b",
    "openai/gpt-oss-120b": "gpt-oss-120b",
    "openai/gpt-oss-20b": "gpt-oss-20b",
    "zai-org/GLM-4.6": "glm-4.6",
    "zai-org/GLM-4.5": "glm-4.5"
};

function getModelColor(modelName) {
    return MODEL_COLORS[modelName] || "#6B7280";
}

function getLabInfo(modelName) {
    const n = modelName.toLowerCase();
    for (let labKey in LAB_CONFIG) {
        const lab = LAB_CONFIG[labKey];
        for (let keyword of lab.keywords) {
            if (n.includes(keyword)) {
                return { key: labKey, ...lab };
            }
        }
    }
    return { key: 'default', name: 'Other', icon: null };
}

function getModelDisplayName(modelName) {
    // Check for explicit display name override first
    if (MODEL_DISPLAY_NAMES[modelName]) {
        return MODEL_DISPLAY_NAMES[modelName];
    }
    // Extract just the model name without org prefix if present
    if (modelName.includes('/')) {
        return modelName.split('/').pop();
    }
    return modelName;
}

function getModelWithIcon(modelName, linkUrl) {
    const lab = getLabInfo(modelName);
    const displayName = getModelDisplayName(modelName);
    let iconHtml = '';
    if (lab.icon) {
        iconHtml = '<img src="' + lab.icon + '" class="h-5 w-auto mr-2 flex-shrink-0" alt="' + lab.name + '" title="' + lab.name + '">';
    }
    if (linkUrl) {
        return iconHtml + '<a href="' + linkUrl + '" class="text-blue-600 font-medium hover:text-blue-700 hover:underline">' + displayName + '</a>';
    }
    return iconHtml + '<span class="font-semibold text-zinc-900">' + displayName + '</span>';
}

async function loadData() {
    try {
        const response = await fetch(DATA_URL);
        if (!response.ok) throw new Error("Failed to load data file: " + DATA_URL);
        const text = await response.text();
        const rawData = text.trim().split('\n').map(line => {
            try { return JSON.parse(line); } catch (e) { return null; }
        }).filter(x => x);
        processData(rawData);
        if (window.renderPage) window.renderPage();
    } catch (err) {
        console.error(err);
        const container = document.querySelector('main');
        if (container) {
            container.innerHTML =
                '<div class="max-w-xl mx-auto mt-12 bg-white rounded-lg border border-zinc-200 p-8 text-center">' +
                '<h2 class="text-xl font-semibold text-red-600 mb-4">Error Loading Data</h2>' +
                '<p class="text-zinc-600 mb-2">Could not fetch <code class="bg-zinc-100 px-1.5 py-0.5 rounded text-sm">' + DATA_URL + '</code>.</p>' +
                '<p class="text-zinc-500 text-sm mb-4"><strong>Note:</strong> If opening locally, you must run a local server (browsers block file:// access).</p>' +
                '<code class="bg-zinc-100 px-3 py-2 rounded text-sm inline-block">python3 -m http.server</code>' +
                '</div>';
        }
    }
}

function getModelType(name) {
    const n = name.toLowerCase();
    for (let k in MODEL_KEYWORDS) {
        if (n.includes(k)) return MODEL_KEYWORDS[k];
    }
    return 'Open Source';
}

function passAtK(n, c, k) {
    if (n === 0) return 0.0;
    const p = c / n;
    return (1.0 - Math.pow(1.0 - p, k)) * 100;
}

function processData(rawData) {
    const cleanedData = rawData.map(item => {
        let res = (item.result || 'fail').toString().toLowerCase();
        let msg = null;
        if (res !== 'success' && item.failures && item.failures.length > 0) {
            msg = item.failures[0].message ? item.failures[0].message.trim() : '';
        }
        return {
            model: item.llmConfig?.model || 'Unknown',
            task: item.name || 'Unknown',
            result: res,
            message: msg
        };
    });

    const grouped = {};
    const allTasks = new Set();

    cleanedData.forEach(item => {
        const m = item.model;
        const t = item.task;
        allTasks.add(t);
        if (!grouped[m]) grouped[m] = {};
        if (!grouped[m][t]) grouped[m][t] = [];
        grouped[m][t].push(item);
    });

    const leaderboard = [];
    const model_details = {};

    for (const model in grouped) {
        const tasksMap = grouped[model];
        const p1s = [];
        const p5s = [];
        const pPow5s = [];
        let totalRuns = 0;
        const mRows = [];

        for (const tName in tasksMap) {
            const items = tasksMap[tName];
            const n = items.length;
            const c = items.filter(i => i.result === 'success').length;
            totalRuns += n;
            p1s.push(passAtK(n, c, 1));
            p5s.push(passAtK(n, c, 5));
            if (n > 0) pPow5s.push(Math.pow(c / n, 5) * 100);

            items.forEach((item, idx) => {
                mRows.push({
                    task: tName,
                    result: item.result,
                    run: idx + 1,
                    message: item.message
                });
            });
        }

        const avgP1 = p1s.length ? p1s.reduce((a, b) => a + b, 0) / p1s.length : 0;
        const avgP5 = p5s.length ? p5s.reduce((a, b) => a + b, 0) / p5s.length : 0;
        const avgPPow5 = pPow5s.length ? pPow5s.reduce((a, b) => a + b, 0) / pPow5s.length : 0;
        const taskCount = Object.keys(tasksMap).length;

        leaderboard.push({
            id: model,
            type: getModelType(model),
            p1: parseFloat(avgP1.toFixed(1)),
            p5: parseFloat(avgP5.toFixed(1)),
            pPow5: parseFloat(avgPPow5.toFixed(1)),
            runs: totalRuns,
            tasks: taskCount
        });

        mRows.sort((a, b) => (a.task > b.task) ? 1 : (a.task === b.task) ? a.run - b.run : -1);
        model_details[model] = mRows;
    }

    const tasks = [];
    const task_details = {};

    allTasks.forEach(tName => {
        let allRes = [];
        for (const m in grouped) {
            if (grouped[m][tName]) {
                allRes = allRes.concat(grouped[m][tName].map(i => i.result));
            }
        }
        const nTotal = allRes.length;
        const cTotal = allRes.filter(r => r === 'success').length;

        tasks.push({
            name: tName,
            p1: parseFloat(passAtK(nTotal, cTotal, 1).toFixed(1)),
            count: nTotal
        });

        const breakdown = [];
        for (const m in grouped) {
            if (grouped[m][tName]) {
                const items = grouped[m][tName];
                const n = items.length;
                const c = items.filter(i => i.result === 'success').length;
                const p1 = passAtK(n, c, 1);
                const runs = items.map((i, idx) => ({ r: idx + 1, val: i.result === 'success' ? 'S' : 'F' }));
                breakdown.push({ model: m, p1: parseFloat(p1.toFixed(1)), runs: runs });
            }
        }
        breakdown.sort((a, b) => b.p1 - a.p1);
        task_details[tName] = breakdown;
    });

    leaderboard.sort((a, b) => b.p5 - a.p5);
    tasks.sort((a, b) => a.p1 - b.p1);

    window.PROCESSED_DATA = { leaderboard, tasks, details: model_details, task_details };
}

function getHue(percentage) { return (percentage / 100) * 120; }

function createMiniBar(val, hue) {
    return '<div class="flex-1 h-2 bg-zinc-100 rounded-full overflow-hidden"><div class="h-full rounded-full" style="width: ' + val + '%; background-color: hsl(' + hue + ', 85%, 40%);"></div></div>';
}

function createBar(val, hue) {
    return '<div class="flex-1 h-2 bg-zinc-100 rounded-full overflow-hidden"><div class="h-full rounded-full" style="width: ' + val + '%; background-color: hsl(' + hue + ', 85%, 40%);"></div></div>';
}

// Bar functions with model-specific colors
function createModelBar(val, modelName) {
    const color = getModelColor(modelName);
    return '<div class="flex-1 h-2 bg-zinc-100 rounded-full overflow-hidden"><div class="h-full rounded-full" style="width: ' + val + '%; background-color: ' + color + ';"></div></div>';
}

function createModelMiniBar(val, modelName) {
    const color = getModelColor(modelName);
    return '<div class="flex-1 h-2 bg-zinc-100 rounded-full overflow-hidden"><div class="h-full rounded-full" style="width: ' + val + '%; background-color: ' + color + ';"></div></div>';
}

function sortTable(table, colIndex) {
    const tbody = table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));
    const header = table.querySelector('th[data-idx=\"' + colIndex + '\"]');
    const isAsc = header.classList.contains('asc');
    const dir = isAsc ? -1 : 1;
    rows.sort((a, b) => {
        const aTxt = a.children[colIndex].innerText.trim();
        const bTxt = b.children[colIndex].innerText.trim();
        const aNum = parseFloat(aTxt.replace(/[^0-9.-]+/g, ""));
        const bNum = parseFloat(bTxt.replace(/[^0-9.-]+/g, ""));
        if (!isNaN(aNum) && !isNaN(bNum) && (aTxt.includes('%') || aTxt.match(/^\\d/))) return (aNum - bNum) * dir;
        return aTxt.localeCompare(bTxt, undefined, { numeric: true }) * dir;
    });
    tbody.innerHTML = '';
    tbody.append(...rows);
    table.querySelectorAll('th').forEach(th => th.classList.remove('asc', 'desc'));
    header.classList.toggle('asc', !isAsc);
    header.classList.toggle('desc', isAsc);
}

document.addEventListener('DOMContentLoaded', () => {
    loadData();
    document.querySelectorAll('th[data-idx]').forEach(th => {
        th.addEventListener('click', () => sortTable(th.closest('table'), th.dataset.idx));
    });
});
