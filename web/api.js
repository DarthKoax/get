class APIClient {
  constructor() {
    this.baseUrl = '/api';
  }

  async _handle(res) {
    if (!res.ok) {
      let msg = res.statusText || 'Request failed';
      try {
        const data = await res.json();
        if (data && data.error) msg = data.error;
      } catch (e) {
        // ignore parse errors
      }
      throw new Error(`HTTP ${res.status}: ${msg}`);
    }
    return res;
  }

  async listShortcuts() {
    const res = await fetch(`${this.baseUrl}/shortcuts`);
    await this._handle(res);
    return await res.json();
  }

  async createShortcut(data) {
    const res = await fetch(`${this.baseUrl}/shortcuts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    await this._handle(res);
    return await res.json();
  }

  async updateShortcut(id, data) {
    const res = await fetch(`${this.baseUrl}/shortcuts/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    await this._handle(res);
    return await res.json();
  }

  async deleteShortcut(id) {
    const res = await fetch(`${this.baseUrl}/shortcuts/${id}`, {
      method: 'DELETE'
    });
    await this._handle(res);
    return await res.json();
  }

  async resolveShortcut(name) {
    const res = await fetch(`${this.baseUrl}/resolve/${encodeURIComponent(name)}`);
    await this._handle(res);
    return await res.json();
  }

  async searchShortcuts(query) {
    const res = await fetch(`${this.baseUrl}/search?q=${encodeURIComponent(query)}`);
    await this._handle(res);
    return await res.json();
  }

  async downloadDatabase() {
    const res = await fetch(`${this.baseUrl}/download`);
    if (!res.ok) {
      let msg = res.statusText || 'Download failed';
      try {
        const data = await res.json();
        if (data && data.error) msg = data.error;
      } catch (e) {
        // ignore
      }
      throw new Error(`HTTP ${res.status}: ${msg}`);
    }
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'shortcuts.db';
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  }
}

window.APIClient = APIClient;
window.apiClient = new APIClient();
