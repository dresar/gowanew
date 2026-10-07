import readline from 'node:readline';

const mcpUrl = process.env.GOWA_MCP_URL || 'http://localhost:3000/mcp';
const deviceId = process.env.GOWA_DEVICE_ID || '';
const basicAuth = process.env.GOWA_BASIC_AUTH || '';

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
  terminal: false
});

async function sendRPC(line) {
  const trimmed = line.trim();
  if (!trimmed) return;

  let parsedReq;
  try {
    parsedReq = JSON.parse(trimmed);
  } catch (err) {
    return;
  }

  const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json, text/event-stream'
  };

  if (deviceId) {
    headers['X-Device-Id'] = deviceId;
  }

  if (basicAuth) {
    headers['Authorization'] = `Basic ${Buffer.from(basicAuth).toString('base64')}`;
  }

  try {
    const res = await fetch(mcpUrl, {
      method: 'POST',
      headers,
      body: trimmed
    });

    if (!res.ok) {
      if (parsedReq.id !== undefined && parsedReq.id !== null) {
        const errResp = {
          jsonrpc: '2.0',
          id: parsedReq.id,
          error: {
            code: -32603,
            message: `HTTP error ${res.status}: ${res.statusText}`
          }
        };
        process.stdout.write(JSON.stringify(errResp) + '\n');
      }
      return;
    }

    const raw = await res.text();
    if (!raw.trim()) return;

    if (raw.startsWith('event:') || raw.startsWith('data:')) {
      for (const streamLine of raw.split('\n')) {
        if (streamLine.startsWith('data:')) {
          const jsonPayload = streamLine.slice(5).trim();
          if (jsonPayload) {
            process.stdout.write(jsonPayload + '\n');
          }
        }
      }
    } else {
      process.stdout.write(raw.trim() + '\n');
    }
  } catch (err) {
    if (parsedReq.id !== undefined && parsedReq.id !== null) {
      const errResp = {
        jsonrpc: '2.0',
        id: parsedReq.id,
        error: {
          code: -32603,
          message: err.message || 'Failed to connect to GoWA MCP server'
        }
      };
      process.stdout.write(JSON.stringify(errResp) + '\n');
    }
  }
}

rl.on('line', (line) => {
  sendRPC(line);
});
