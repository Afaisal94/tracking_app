import { api } from '../api/client';

export type QueuedPoint = {
  lat: number;
  lng: number;
  accuracy?: number | null;
  speed?: number | null;
  altitude?: number | null;
  timestamp: string;
  session_id?: string;
};

class QueueService {
  private queue: QueuedPoint[] = [];
  private flushing = false;

  enqueue(p: QueuedPoint) {
    this.queue.push(p);
    // keep max 2000 points in memory; older points dropped (should be rare)
    if (this.queue.length > 2000) this.queue.shift();
  }

  size() {
    return this.queue.length;
  }

  async flush() {
    if (this.flushing || this.queue.length === 0) return;
    this.flushing = true;
    const batch = [...this.queue];
    try {
      await api.post('/api/v1/tracking/points/batch', { points: batch });
      // remove flushed items
      this.queue.splice(0, batch.length);
    } catch (e) {
      console.warn('[Queue] flush failed', e);
    } finally {
      this.flushing = false;
    }
  }

  clear() {
    this.queue = [];
  }
}

export const queueService = new QueueService();
