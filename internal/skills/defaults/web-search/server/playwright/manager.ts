/**
 * Playwright Manager - Manages browser connections and page sessions using Playwright
 */

import { chromium, Browser, Page, BrowserContext } from 'playwright-core';
import { v4 as uuidv4 } from 'uuid';

export interface Connection {
  id: string;
  browser: Browser;
  context: BrowserContext;
  pages: Map<string, Page>;
  connectedAt: number;
}

export class PlaywrightManager {
  private connections: Map<string, Connection> = new Map();

  private isConnectionAlive(conn: Connection): boolean {
    try {
      if (!conn.browser.isConnected()) {
        return false;
      }

      // Accessing pages throws when context is already closed.
      conn.context.pages();
      return true;
    } catch {
      return false;
    }
  }

  private pruneDeadConnections(): void {
    for (const [connectionId, conn] of this.connections.entries()) {
      if (!this.isConnectionAlive(conn)) {
        console.warn(`[Playwright] Removing stale connection: ${connectionId}`);
        this.connections.delete(connectionId);
      }
    }
  }

  /**
   * Get CDP WebSocket debugger URL
   */
  private async getCDPWebSocketUrl(port: number): Promise<string> {
    const response = await fetch(`http://127.0.0.1:${port}/json/version`);
    const data = await response.json() as { webSocketDebuggerUrl: string };
    return data.webSocketDebuggerUrl;
  }

  /**
   * Connect to Chrome via CDP using Playwright
   */
  async connectToCDP(port: number = 9222): Promise<string> {
    try {
      console.log(`[Playwright] Connecting to CDP on port ${port}`);

      const wsUrl = await this.getCDPWebSocketUrl(port);
      console.log(`[Playwright] CDP WebSocket URL: ${wsUrl}`);

      const browser = await chromium.connectOverCDP(wsUrl);
      console.log(`[Playwright] Connected to browser`);

      // Get or create browser context
      const contexts = browser.contexts();
      let context: BrowserContext;

      if (contexts.length === 0) {
        console.log(`[Playwright] No existing context, creating new one`);
        context = await browser.newContext();
      } else {
        console.log(`[Playwright] Using existing context`);
        context = contexts[0];
      }

      const connectionId = uuidv4();
      const connection: Connection = {
        id: connectionId,
        browser,
        context,
        pages: new Map(),
        connectedAt: Date.now()
      };

      this.connections.set(connectionId, connection);

      console.log(`[Playwright] Connection established: ${connectionId}`);
      return connectionId;
    } catch (error) {
      console.error(`[Playwright] Failed to connect to CDP:`, error);
      throw new Error(`Failed to connect to CDP: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  /**
   * Get or create a page for the connection
   */
  async getPage(connectionId: string): Promise<Page> {
    this.pruneDeadConnections();

    const conn = this.connections.get(connectionId);
    if (!conn) {
      throw new Error(`Connection not found: ${connectionId}`);
    }

    if (!this.isConnectionAlive(conn)) {
      this.connections.delete(connectionId);
      throw new Error(`Connection not active: ${connectionId}`);
    }

    // Check for existing pages in the context
    const contextPages = conn.context.pages().filter(page => !page.isClosed());

    if (contextPages.length === 0) {
      console.log(`[Playwright] No existing pages, creating new page`);
      try {
        const page = await conn.context.newPage();
        conn.pages.set(page.url(), page);
        return page;
      } catch (error) {
        this.connections.delete(connectionId);
        throw new Error(`Connection became invalid: ${error instanceof Error ? error.message : String(error)}`);
      }
    }

    // Return the first page (main page)
    const page = contextPages[0];
    console.log(`[Playwright] Using existing page: ${page.url()}`);
    return page;
  }

  /**
   * Create a new page in the connection
   */
  async createPage(connectionId: string): Promise<Page> {
    this.pruneDeadConnections();

    const conn = this.connections.get(connectionId);
    if (!conn) {
      throw new Error(`Connection not found: ${connectionId}`);
    }

    if (!this.isConnectionAlive(conn)) {
      this.connections.delete(connectionId);
      throw new Error(`Connection not active: ${connectionId}`);
    }

    console.log(`[Playwright] Creating new page for connection ${connectionId}`);
    try {
      const page = await conn.context.newPage();
      conn.pages.set(page.url(), page);
      return page;
    } catch (error) {
      this.connections.delete(connectionId);
      throw new Error(`Connection became invalid: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  /**
   * Close a specific page
   */
  async closePage(connectionId: string, page: Page): Promise<void> {
    const conn = this.connections.get(connectionId);
    if (!conn) {
      throw new Error(`Connection not found: ${connectionId}`);
    }

    console.log(`[Playwright] Closing page: ${page.url()}`);
    await page.close();
    conn.pages.delete(page.url());
  }

  /**
   * Get connection info
   */
  getConnection(connectionId: string): Connection | undefined {
    this.pruneDeadConnections();
    return this.connections.get(connectionId);
  }

  /**
   * List all active connections
   */
  listConnections(): Array<{ id: string; connectedAt: number; pageCount: number }> {
    this.pruneDeadConnections();

    return Array.from(this.connections.values()).map(conn => ({
      id: conn.id,
      connectedAt: conn.connectedAt,
      pageCount: conn.context.pages().filter(page => !page.isClosed()).length
    }));
  }

  /**
   * Disconnect from browser
   */
  async disconnect(connectionId: string): Promise<void> {
    const conn = this.connections.get(connectionId);
    if (!conn) {
      console.warn(`[Playwright] Connection not found: ${connectionId}`);
      return;
    }

    console.log(`[Playwright] Disconnecting connection: ${connectionId}`);

    try {
      // Close all pages
      const pages = conn.context.pages();
      for (const page of pages) {
        try {
          await page.close();
        } catch (error) {
          console.warn(`[Playwright] Failed to close page:`, error);
        }
      }

      // Close context (if we created it)
      try {
        await conn.context.close();
      } catch (error) {
        console.warn(`[Playwright] Failed to close context:`, error);
      }

      // Close browser connection
      await conn.browser.close();
      console.log(`[Playwright] Browser connection closed: ${connectionId}`);
    } catch (error) {
      console.error(`[Playwright] Error during disconnect:`, error);
    } finally {
      this.connections.delete(connectionId);
    }
  }

  /**
   * Disconnect all connections
   */
  async disconnectAll(): Promise<void> {
    console.log(`[Playwright] Disconnecting all connections (${this.connections.size})`);
    const connectionIds = Array.from(this.connections.keys());

    for (const connectionId of connectionIds) {
      await this.disconnect(connectionId);
    }
  }

  /**
   * Check if connection exists and is valid
   */
  isConnected(connectionId: string): boolean {
    this.pruneDeadConnections();

    const conn = this.connections.get(connectionId);
    if (!conn) {
      return false;
    }

    return this.isConnectionAlive(conn);
  }

  /**
   * Get connection count
   */
  getConnectionCount(): number {
    this.pruneDeadConnections();
    return this.connections.size;
  }
}
