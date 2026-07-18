import React, { useState, useEffect } from "react";
import { 
  Database, Network, Layers, RotateCw, Play, Trash2, Plus, 
  FileJson, ShieldCheck, History, AlertTriangle, Send, CheckCircle2, 
  RefreshCw, RefreshCcw, HelpCircle, AlertCircle
} from "lucide-react";

interface EventItem {
  id: string;
  streamId: string;
  type: string;
  version: number;
  data: any;
  createdAt: string;
}

interface SnapshotItem {
  streamId: string;
  version: number;
  state: any;
  createdAt: string;
}

interface TopicMetadata {
  name: string;
  partitions: number;
  replicationFactor: number;
  messageCount: number;
}

interface SchemaItem {
  eventType: string;
  version: number;
  requiredFields: string[];
}

interface RetryItem {
  eventId: string;
  topic: string;
  retryCount: number;
  nextRetryAt: number;
  delayMs: number;
  lastError: string;
  event: EventItem;
}

interface DLQItem {
  id: string;
  topic: string;
  event: EventItem;
  reason: string;
  failedAt: string;
}

interface ProjectionModel {
  id: string;
  amount: number;
  status: string;
  version: number;
  updatedAt: string;
}

export default function EventDrivenDashboard() {
  // -----------------------------------------------------------------
  // STATE MANAGEMENT
  // -----------------------------------------------------------------
  
  // 1. EventStoreDB State
  const [events, setEvents] = useState<EventItem[]>([
    { id: "evt_382109", streamId: "transaction-stream-tx_9921", type: "TransactionCreated", version: 1, data: { id: "tx_9921", amount: 15200.50, created_by: "principal@nexuscore.io" }, createdAt: new Date(Date.now() - 3600000).toISOString() },
    { id: "evt_382110", streamId: "transaction-stream-tx_9921", type: "TransactionProcessed", version: 2, data: { id: "tx_9921", amount: 15200.50, status: "APPROVED", created_by: "principal@nexuscore.io" }, createdAt: new Date(Date.now() - 3590000).toISOString() },
    { id: "evt_382111", streamId: "transaction-stream-tx_9922", type: "TransactionCreated", version: 1, data: { id: "tx_9922", amount: 89.90, created_by: "user_enterprise_01" }, createdAt: new Date(Date.now() - 900000).toISOString() },
    { id: "evt_382112", streamId: "transaction-stream-tx_9922", type: "TransactionProcessed", version: 2, data: { id: "tx_9922", amount: 89.90, status: "APPROVED", created_by: "user_enterprise_01" }, createdAt: new Date(Date.now() - 890000).toISOString() },
  ]);

  const [snapshots, setSnapshots] = useState<SnapshotItem[]>([
    { streamId: "transaction-stream-tx_9921", version: 2, state: { id: "tx_9921", amount: 15200.50, status: "APPROVED" }, createdAt: new Date(Date.now() - 3500000).toISOString() }
  ]);

  const [selectedStream, setSelectedStream] = useState<string>("transaction-stream-tx_9921");

  // 2. Schema Registry State
  const [schemas, setSchemas] = useState<SchemaItem[]>([
    { eventType: "TransactionCreated", version: 1, requiredFields: ["id", "amount", "created_by"] },
    { eventType: "TransactionCreated", version: 2, requiredFields: ["id", "amount", "created_by", "tenant_id"] },
    { eventType: "TransactionProcessed", version: 1, requiredFields: ["id", "status"] }
  ]);

  const [newSchemaType, setNewSchemaType] = useState<string>("TransactionRefunded");
  const [newSchemaVersion, setNewSchemaVersion] = useState<number>(1);
  const [newSchemaFields, setNewSchemaFields] = useState<string>("id, amount, refund_reason");

  // 3. Kafka Cluster & Topic Management State
  const [topics, setTopics] = useState<TopicMetadata[]>([
    { name: "transaction-events", partitions: 3, replicationFactor: 2, messageCount: 4 },
    { name: "transaction-events-retry", partitions: 3, replicationFactor: 2, messageCount: 0 },
    { name: "transaction-events-dlq", partitions: 1, replicationFactor: 3, messageCount: 0 }
  ]);

  const [newTopicName, setNewTopicName] = useState<string>("");
  const [newTopicPartitions, setNewTopicPartitions] = useState<number>(3);

  // 4. Retry and Dead Letter Queues (DLQ) State
  const [retryQueue, setRetryQueue] = useState<RetryItem[]>([]);
  const [dlq, setDlq] = useState<DLQItem[]>([]);

  // 5. CQRS Read Model Projections Store
  const [readModel, setReadModel] = useState<ProjectionModel[]>([
    { id: "tx_9921", amount: 15200.50, status: "APPROVED", version: 2, updatedAt: new Date(Date.now() - 3590000).toISOString() },
    { id: "tx_9922", amount: 89.90, status: "APPROVED", version: 2, updatedAt: new Date(Date.now() - 890000).toISOString() }
  ]);

  // 6. Interactive Command States
  const [cmdId, setCmdId] = useState<string>(`tx_${Math.random().toString(36).substring(2, 7)}`);
  const [cmdAmount, setCmdAmount] = useState<number>(450.00);
  const [cmdUser, setCmdUser] = useState<string>("principal@nexuscore.io");
  const [cmdTenant, setCmdTenant] = useState<string>("tenant_global_core");
  const [cmdSchemaVersion, setCmdSchemaVersion] = useState<number>(1);
  const [simulateFailure, setSimulateFailure] = useState<boolean>(false);
  const [processingState, setProcessingState] = useState<string>("idle"); // idle, validating, appending, publishing, routing, success

  // -----------------------------------------------------------------
  // OPERATIONS / ACTIONS
  // -----------------------------------------------------------------

  // Generate random TX ID helper
  const handleRegenId = () => {
    setCmdId(`tx_${Math.random().toString(36).substring(2, 7)}`);
  };

  // 1. Topic Management Actions
  const handleCreateTopic = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTopicName.trim()) return;

    const topicExists = topics.some(t => t.name === newTopicName);
    if (topicExists) {
      alert(`Topic '${newTopicName}' already exists in Kafka cluster metadata!`);
      return;
    }

    setTopics(prev => [...prev, {
      name: newTopicName.trim(),
      partitions: newTopicPartitions,
      replicationFactor: 2,
      messageCount: 0
    }]);
    setNewTopicName("");
  };

  const handleDeleteTopic = (name: string) => {
    if (name.startsWith("transaction-events")) {
      alert("System core topics cannot be deleted to prevent cluster crash.");
      return;
    }
    setTopics(prev => prev.filter(t => t.name !== name));
  };

  // 2. Schema Registry Actions
  const handleRegisterSchema = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newSchemaType.trim()) return;

    const fieldsArray = newSchemaFields.split(",").map(f => f.trim()).filter(Boolean);
    setSchemas(prev => [...prev, {
      eventType: newSchemaType.trim(),
      version: newSchemaVersion,
      requiredFields: fieldsArray
    }]);
    setNewSchemaType("");
    setNewSchemaFields("id, amount");
  };

  // 3. Command Execution Simulator (CQRS + Event Sourcing Pipeline)
  const handleDispatchCommand = async () => {
    setProcessingState("validating");
    
    // Step A: Schema Validation Checks
    const activeSchema = schemas.find(s => s.eventType === "TransactionCreated" && s.version === cmdSchemaVersion);
    const dataPayload: Record<string, any> = {
      id: cmdId,
      amount: cmdAmount,
      created_by: cmdUser,
    };
    if (cmdSchemaVersion === 2) {
      dataPayload.tenant_id = cmdTenant;
    }

    await new Promise(r => setTimeout(r, 600)); // Network validation latency

    if (activeSchema) {
      // Check required fields
      const missing = activeSchema.requiredFields.filter(f => dataPayload[f] === undefined || dataPayload[f] === "");
      if (missing.length > 0) {
        setProcessingState("idle");
        alert(`Schema registry error: validation failed for TransactionCreated v${cmdSchemaVersion}. Missing fields: ${missing.join(", ")}`);
        return;
      }
    }

    // Step B: Append to EventStoreDB (Immature Aggregate Stream Initialization)
    setProcessingState("appending");
    await new Promise(r => setTimeout(r, 600));

    const streamId = `transaction-stream-${cmdId}`;
    
    // Verify stream duplication
    const duplicate = events.some(e => e.streamId === streamId);
    if (duplicate) {
      setProcessingState("idle");
      alert(`Concurrency conflict: Stream '${streamId}' already exists in EventStoreDB!`);
      return;
    }

    const createdEvent: EventItem = {
      id: `evt_${Math.floor(100000 + Math.random() * 900000)}`,
      streamId,
      type: "TransactionCreated",
      version: 1,
      data: dataPayload,
      createdAt: new Date().toISOString()
    };

    const status = cmdAmount > 1000000 ? "REJECTED" : "APPROVED";
    const processedEvent: EventItem = {
      id: `evt_${Math.floor(100000 + Math.random() * 900000)}`,
      streamId,
      type: "TransactionProcessed",
      version: 2,
      data: { id: cmdId, amount: cmdAmount, status, created_by: cmdUser },
      createdAt: new Date(Date.now() + 50).toISOString()
    };

    setEvents(prev => [...prev, createdEvent, processedEvent]);
    setSelectedStream(streamId);

    // Increment topic counters
    setTopics(prev => prev.map(t => t.name === "transaction-events" ? { ...t, messageCount: t.messageCount + 2 } : t));

    // Step C: Publish Event Packet to Kafka Cluster Ingress Broker
    setProcessingState("publishing");
    await new Promise(r => setTimeout(r, 500));

    // Handle consumer failures and Retry/DLQ pipelines
    if (simulateFailure) {
      setProcessingState("routing");
      await new Promise(r => setTimeout(r, 600));

      // Route to Retry Queue
      const retryItem: RetryItem = {
        eventId: processedEvent.id,
        topic: "transaction-events",
        retryCount: 1,
        nextRetryAt: Date.now() + 5000, // retry in 5s
        delayMs: 5000,
        lastError: "handshake timeout with ledger db projection server",
        event: processedEvent
      };

      setRetryQueue(prev => [...prev, retryItem]);
      setProcessingState("idle");
      setTopics(prev => prev.map(t => t.name === "transaction-events-retry" ? { ...t, messageCount: t.messageCount + 1 } : t));
    } else {
      // Normal flow - Project update synchronously
      setProcessingState("success");
      await new Promise(r => setTimeout(r, 400));

      // Append directly to separation read projection query model store
      const projection: ProjectionModel = {
        id: cmdId,
        amount: cmdAmount,
        status,
        version: 2,
        updatedAt: new Date().toISOString()
      };
      setReadModel(prev => [projection, ...prev]);
      setProcessingState("idle");
      handleRegenId();
    }
  };

  // 4. Retry Loop Simulator Countdown (Tick loop every second)
  useEffect(() => {
    const timer = setInterval(() => {
      const now = Date.now();
      setRetryQueue(prev => {
        const expired = prev.filter(r => now >= r.nextRetryAt);
        const active = prev.filter(r => now < r.nextRetryAt);

        expired.forEach(item => {
          if (item.retryCount >= 3) {
            // Exceeded max retries, move to Dead Letter Queue (DLQ)
            const dlqItem: DLQItem = {
              id: `dlq_${Math.floor(100000 + Math.random() * 900000)}`,
              topic: item.topic,
              event: item.event,
              reason: `max retries (3) exhausted. Root error: ${item.lastError}`,
              failedAt: new Date().toISOString()
            };
            setDlq(d => [...d, dlqItem]);
            setTopics(t => t.map(topic => topic.name === "transaction-events-dlq" ? { ...topic, messageCount: topic.messageCount + 1 } : topic));
          } else {
            // Schedule next exponential retry: Delay = Delay * 2
            const nextDelay = item.delayMs * 2;
            const nextRetry: RetryItem = {
              ...item,
              retryCount: item.retryCount + 1,
              nextRetryAt: Date.now() + nextDelay,
              delayMs: nextDelay,
              lastError: `network routing socket refused (attempt ${item.retryCount + 1}/3)`
            };
            active.push(nextRetry);
          }
        });

        return active;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, []);

  // 5. Redeliver DLQ Message Action
  const handleRedeliverDLQ = (dlqId: string) => {
    const item = dlq.find(d => d.id === dlqId);
    if (!item) return;

    // Simulate successful redelivery (fix connectivity simulation)
    const tx = item.event.data;
    const isRejected = tx.amount > 1000000;
    const status = isRejected ? "REJECTED" : "APPROVED";

    const projection: ProjectionModel = {
      id: tx.id,
      amount: tx.amount,
      status: status,
      version: item.event.version,
      updatedAt: new Date().toISOString()
    };

    setReadModel(prev => {
      if (prev.some(m => m.id === tx.id)) {
        return prev.map(m => m.id === tx.id ? projection : m);
      }
      return [projection, ...prev];
    });

    setDlq(prev => prev.filter(d => d.id !== dlqId));
    alert(`DLQ message '${item.event.id}' was successfully routed manually to partition-consumer projection model!`);
  };

  const handlePurgeDLQ = (dlqId: string) => {
    setDlq(prev => prev.filter(d => d.id !== dlqId));
  };

  // 6. Snapshot Aggregation Execution
  const handleTakeSnapshot = (streamId: string) => {
    const streamEvents = events.filter(e => e.streamId === streamId);
    if (streamEvents.length === 0) return;

    const latestVersion = streamEvents[streamEvents.length - 1].version;
    const baseCreated = streamEvents.find(e => e.type === "TransactionCreated");
    const baseProcessed = streamEvents.find(e => e.type === "TransactionProcessed");

    const amount = baseCreated ? baseCreated.data.amount : 0;
    const status = baseProcessed ? baseProcessed.data.status : "PENDING";

    const snapshot: SnapshotItem = {
      streamId,
      version: latestVersion,
      state: { id: streamId.replace("transaction-stream-", ""), amount, status },
      createdAt: new Date().toISOString()
    };

    setSnapshots(prev => {
      const filtered = prev.filter(s => s.streamId !== streamId);
      return [...filtered, snapshot];
    });

    alert(`State-backed Snapshot taken successfully for stream '${streamId}' at version ${latestVersion}!`);
  };

  // 7. Event Replay Mechanism
  const handleReplayEvents = (streamId: string) => {
    const streamEvents = events.filter(e => e.streamId === streamId);
    if (streamEvents.length === 0) return;

    // Simulating clear and rebuild
    const txId = streamId.replace("transaction-stream-", "");
    
    // Clear the specific model
    setReadModel(prev => prev.filter(m => m.id !== txId));

    setTimeout(() => {
      // Rebuild state step-by-step
      const baseCreated = streamEvents.find(e => e.type === "TransactionCreated");
      const baseProcessed = streamEvents.find(e => e.type === "TransactionProcessed");

      const amount = baseCreated ? baseCreated.data.amount : 0;
      const status = baseProcessed ? baseProcessed.data.status : "PENDING";

      const projection: ProjectionModel = {
        id: txId,
        amount,
        status,
        version: streamEvents[streamEvents.length - 1].version,
        updatedAt: new Date().toISOString()
      };

      setReadModel(prev => [projection, ...prev]);
      alert(`Event Replay successfully finished for stream '${streamId}'! Re-evaluated ${streamEvents.length} sequential events, rebuilding read projection model successfully.`);
    }, 800);
  };

  // Group events by streamId
  const streamsList = Array.from(new Set(events.map(e => e.streamId)));

  return (
    <div className="space-y-6 text-slate-200" id="nexuscore-eventdriven-root">
      {/* Bento Grid layout */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        
        {/* ==========================================
            1. EVENTSTOREDB IMMUTABLE LOG
           ========================================== */}
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 flex flex-col h-[520px]">
          <div className="flex items-center justify-between border-b border-slate-900 pb-3 mb-4">
            <div className="flex items-center gap-2">
              <Database className="text-emerald-400" size={18} />
              <h3 className="font-display font-semibold text-white text-sm">EventStoreDB Log</h3>
            </div>
            <span className="text-[10px] font-mono bg-emerald-950/30 text-emerald-400 border border-emerald-900/40 px-2 py-0.5 rounded-full font-bold">
              OCC Active
            </span>
          </div>

          <div className="grid grid-cols-3 gap-2 mb-3">
            <div className="col-span-2 space-y-1">
              <label className="text-[9px] text-slate-500 uppercase font-bold tracking-wider">Aggregate Streams</label>
              <select 
                value={selectedStream}
                onChange={e => setSelectedStream(e.target.value)}
                className="w-full bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-300 font-mono focus:outline-none"
              >
                {streamsList.map(s => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
            </div>
            <div className="flex items-end gap-1">
              <button 
                onClick={() => handleTakeSnapshot(selectedStream)}
                title="Create Snapshot"
                className="flex-1 bg-slate-900 border border-slate-800 hover:bg-slate-800 p-2 rounded-lg flex justify-center text-indigo-400 hover:text-indigo-300 transition-colors"
              >
                <Layers size={14} />
              </button>
              <button 
                onClick={() => handleReplayEvents(selectedStream)}
                title="Trigger Event Replay"
                className="flex-1 bg-slate-900 border border-slate-800 hover:bg-slate-800 p-2 rounded-lg flex justify-center text-amber-400 hover:text-amber-300 transition-colors"
              >
                <RefreshCcw size={14} />
              </button>
            </div>
          </div>

          {/* Stream Events Scrollbox */}
          <div className="flex-1 overflow-y-auto space-y-3 pr-1">
            {events.filter(e => e.streamId === selectedStream).map(e => (
              <div key={e.id} className="bg-slate-900/50 border border-slate-900 rounded-lg p-3 space-y-2 hover:border-slate-800 transition-colors">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] font-mono text-emerald-400 font-bold bg-emerald-950/20 px-1.5 py-0.5 rounded">
                    v{e.version}
                  </span>
                  <span className="text-[10px] font-mono text-slate-500">{e.id}</span>
                </div>
                <div>
                  <div className="text-[11px] font-mono font-bold text-white mb-1">{e.type}</div>
                  <pre className="text-[10px] font-mono text-slate-400 bg-slate-950 p-2 rounded max-h-[100px] overflow-y-auto">
                    {JSON.stringify(e.data, null, 2)}
                  </pre>
                </div>
                <div className="text-[8px] font-mono text-slate-600 text-right">
                  {new Date(e.createdAt).toLocaleTimeString()}
                </div>
              </div>
            ))}

            {/* Render Snapshots if exist */}
            {snapshots.some(s => s.streamId === selectedStream) && (
              <div className="border-t border-indigo-950/40 pt-3 mt-2">
                <div className="text-[10px] text-indigo-400 font-bold font-sans uppercase mb-1.5 flex items-center gap-1">
                  <Layers size={11} /> Aggregation Snapshot
                </div>
                {snapshots.filter(s => s.streamId === selectedStream).map((s, idx) => (
                  <div key={idx} className="bg-indigo-950/10 border border-indigo-900/30 rounded-lg p-2.5 space-y-1.5">
                    <div className="flex justify-between text-[9px] font-mono">
                      <span className="text-indigo-400 font-bold">Snapshot State @ v{s.version}</span>
                      <span className="text-slate-500">{new Date(s.createdAt).toLocaleTimeString()}</span>
                    </div>
                    <pre className="text-[10px] font-mono text-slate-300 bg-slate-950/50 p-2 rounded">
                      {JSON.stringify(s.state, null, 2)}
                    </pre>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* ==========================================
            2. KAFKA CLUSTER & TOPIC MANAGER
           ========================================== */}
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 flex flex-col h-[520px]">
          <div className="flex items-center justify-between border-b border-slate-900 pb-3 mb-4">
            <div className="flex items-center gap-2">
              <Network className="text-indigo-400" size={18} />
              <h3 className="font-display font-semibold text-white text-sm">Kafka Cluster & Topics</h3>
            </div>
            <span className="text-[10px] font-mono bg-indigo-950/30 text-indigo-400 border border-indigo-900/40 px-2 py-0.5 rounded-full font-bold">
              3 Nodes
            </span>
          </div>

          {/* Topics List */}
          <div className="flex-1 overflow-y-auto space-y-3 mb-4 pr-1">
            <div className="text-[10px] text-slate-500 uppercase font-bold tracking-wider mb-1">Active Message Topics</div>
            {topics.map(t => (
              <div key={t.name} className="bg-slate-900/40 border border-slate-900 rounded-lg p-3 flex items-center justify-between group hover:border-slate-800 transition-colors">
                <div className="space-y-1">
                  <div className="text-xs font-mono font-bold text-slate-200">{t.name}</div>
                  <div className="flex gap-2.5 text-[9px] font-mono text-slate-500">
                    <span>Partitions: <span className="text-slate-400 font-bold">{t.partitions}</span></span>
                    <span>•</span>
                    <span>Replicas: <span className="text-slate-400 font-bold">{t.replicationFactor}</span></span>
                    <span>•</span>
                    <span>Messages: <span className="text-indigo-400 font-bold">{t.messageCount}</span></span>
                  </div>
                </div>
                <button 
                  onClick={() => handleDeleteTopic(t.name)}
                  className="text-slate-500 hover:text-rose-400 p-1 rounded-md opacity-0 group-hover:opacity-100 hover:bg-rose-950/20 transition-all"
                  title="Delete Topic"
                >
                  <Trash2 size={13} />
                </button>
              </div>
            ))}
          </div>

          {/* Create Topic Form */}
          <form onSubmit={handleCreateTopic} className="border-t border-slate-900 pt-3 space-y-2.5">
            <div className="text-[10px] text-slate-500 uppercase font-bold tracking-wider">Provision New Topic</div>
            <div className="grid grid-cols-3 gap-2">
              <input 
                type="text"
                placeholder="topic-name-events"
                value={newTopicName}
                onChange={e => setNewTopicName(e.target.value)}
                className="col-span-2 bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500/50"
              />
              <input 
                type="number"
                min="1"
                max="12"
                value={newTopicPartitions}
                onChange={e => setNewTopicPartitions(parseInt(e.target.value) || 3)}
                className="bg-slate-900 border border-slate-800 rounded-lg px-2 py-1.5 text-xs text-slate-200 text-center focus:outline-none focus:border-indigo-500/50"
                title="Partitions"
              />
            </div>
            <button 
              type="submit"
              className="w-full bg-indigo-600 hover:bg-indigo-700 py-1.5 rounded-lg text-xs font-bold text-white transition-colors flex items-center justify-center gap-1 shadow-sm"
            >
              <Plus size={14} /> PROVISION TOPIC
            </button>
          </form>
        </div>

        {/* ==========================================
            3. SCHEMA REGISTRY & EVENT VERSIONING
           ========================================== */}
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 flex flex-col h-[520px]">
          <div className="flex items-center justify-between border-b border-slate-900 pb-3 mb-4">
            <div className="flex items-center gap-2">
              <ShieldCheck className="text-amber-400" size={18} />
              <h3 className="font-display font-semibold text-white text-sm">Schema Registry</h3>
            </div>
            <span className="text-[10px] font-mono bg-amber-950/30 text-amber-400 border border-amber-900/40 px-2 py-0.5 rounded-full font-bold">
              Dynamic Guard
            </span>
          </div>

          {/* Schemas List */}
          <div className="flex-1 overflow-y-auto space-y-3 mb-4 pr-1">
            <div className="text-[10px] text-slate-500 uppercase font-bold tracking-wider mb-1">Registered Event Schemas</div>
            {schemas.map((s, idx) => (
              <div key={idx} className="bg-slate-900/40 border border-slate-900 rounded-lg p-3 space-y-1.5 hover:border-slate-800 transition-colors">
                <div className="flex justify-between items-center">
                  <span className="text-xs font-mono font-bold text-amber-400">{s.eventType}</span>
                  <span className="text-[9px] font-mono bg-slate-950 border border-slate-800 text-slate-400 px-1.5 py-0.5 rounded">
                    Version {s.version}
                  </span>
                </div>
                <div className="text-[10px] font-mono text-slate-500 flex flex-wrap gap-1">
                  {s.requiredFields.map(f => (
                    <span key={f} className="bg-slate-950/50 border border-slate-900 rounded px-1.5 py-0.5 text-slate-300">
                      {f}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>

          {/* Register Schema Form */}
          <form onSubmit={handleRegisterSchema} className="border-t border-slate-900 pt-3 space-y-2">
            <div className="text-[10px] text-slate-500 uppercase font-bold tracking-wider">Register New Event Schema</div>
            <div className="grid grid-cols-3 gap-2">
              <input 
                type="text"
                placeholder="EventType"
                value={newSchemaType}
                onChange={e => setNewSchemaType(e.target.value)}
                className="col-span-2 bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-amber-500/40"
              />
              <input 
                type="number"
                min="1"
                value={newSchemaVersion}
                onChange={e => setNewSchemaVersion(parseInt(e.target.value) || 1)}
                className="bg-slate-900 border border-slate-800 rounded-lg px-2 py-1.5 text-xs text-slate-200 text-center focus:outline-none focus:border-amber-500/40"
                title="Version"
              />
            </div>
            <input 
              type="text"
              placeholder="required_field_1, required_field_2"
              value={newSchemaFields}
              onChange={e => setNewSchemaFields(e.target.value)}
              className="w-full bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-amber-500/40"
            />
            <button 
              type="submit"
              className="w-full bg-amber-600 hover:bg-amber-700 py-1.5 rounded-lg text-xs font-bold text-white transition-colors flex items-center justify-center gap-1 shadow-sm"
            >
              <Plus size={14} /> ENFORCE SCHEMA
            </button>
          </form>
        </div>

      </div>

      {/* =========================================================
          4. RETRY PIPELINES AND DEAD LETTER QUEUE (DLQ)
         ========================================================= */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        
        {/* RETRY PIPELINE */}
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 flex flex-col h-[320px]">
          <div className="flex items-center justify-between border-b border-slate-900 pb-3 mb-3">
            <div className="flex items-center gap-2">
              <RefreshCw className="text-indigo-400 animate-spin-slow" size={16} />
              <h3 className="font-display font-semibold text-white text-xs">Exponential Backoff Retry Queue</h3>
            </div>
            <span className="text-[10px] font-mono text-slate-500 font-bold">
              Base: 5000ms • Max Attempt: 3
            </span>
          </div>

          <div className="flex-1 overflow-y-auto min-h-0">
            {retryQueue.length === 0 ? (
              <div className="h-full flex flex-col items-center justify-center text-slate-600 text-center text-xs gap-1.5">
                <CheckCircle2 size={24} className="text-emerald-500/80" />
                <span>Retry queue clear. All partition consumer channels healthy.</span>
              </div>
            ) : (
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr className="border-b border-slate-900 text-slate-500 font-bold font-sans">
                    <th className="pb-2">Event ID</th>
                    <th className="pb-2">Target Topic</th>
                    <th className="pb-2 text-center">Attempt</th>
                    <th className="pb-2 text-right">Trigger Countdown</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-900 font-mono">
                  {retryQueue.map(item => {
                    const secondsLeft = Math.max(0, Math.round((item.nextRetryAt - Date.now()) / 1000));
                    return (
                      <tr key={item.eventId} className="text-slate-300">
                        <td className="py-2">{item.eventId}</td>
                        <td className="py-2 text-indigo-400">{item.topic}</td>
                        <td className="py-2 text-center text-amber-400 font-bold">{item.retryCount}/3</td>
                        <td className="py-2 text-right text-emerald-400 font-bold">
                          {secondsLeft > 0 ? `T-${secondsLeft}s` : "RETRIES RUNNING..."}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        </div>

        {/* DEAD LETTER QUEUE (DLQ) */}
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 flex flex-col h-[320px]">
          <div className="flex items-center justify-between border-b border-slate-900 pb-3 mb-3">
            <div className="flex items-center gap-2">
              <AlertTriangle className="text-rose-400" size={16} />
              <h3 className="font-display font-semibold text-white text-xs">Dead Letter Queue (DLQ) Quarantine</h3>
            </div>
            <span className="text-[10px] font-mono text-rose-400 font-bold bg-rose-950/20 px-2 py-0.5 rounded border border-rose-900/30">
              {dlq.length} Trapped Packs
            </span>
          </div>

          <div className="flex-1 overflow-y-auto min-h-0">
            {dlq.length === 0 ? (
              <div className="h-full flex flex-col items-center justify-center text-slate-600 text-center text-xs gap-1.5">
                <CheckCircle2 size={24} className="text-emerald-500/80" />
                <span>Dead letter queue clear. No quarantined message frames.</span>
              </div>
            ) : (
              <div className="space-y-3">
                {dlq.map(item => (
                  <div key={item.id} className="bg-slate-900/50 border border-rose-950/30 rounded-lg p-3 space-y-2">
                    <div className="flex justify-between items-center text-[10px] font-mono">
                      <span className="text-rose-400 font-bold flex items-center gap-1">
                        <AlertCircle size={12} /> {item.id}
                      </span>
                      <span className="text-slate-500">Quarantined from: <span className="text-indigo-400">{item.topic}</span></span>
                    </div>
                    <div className="text-[10px] font-mono text-slate-400 leading-relaxed">
                      <span className="text-slate-500 font-bold block">Rejection Reason:</span>
                      {item.reason}
                    </div>
                    <div className="flex gap-2 justify-end pt-1">
                      <button 
                        onClick={() => handlePurgeDLQ(item.id)}
                        className="text-[10px] font-sans font-bold bg-slate-950 border border-slate-800 hover:bg-slate-800 text-slate-400 hover:text-white px-2.5 py-1 rounded transition-colors"
                      >
                        PURGE
                      </button>
                      <button 
                        onClick={() => handleRedeliverDLQ(item.id)}
                        className="text-[10px] font-sans font-bold bg-emerald-600 hover:bg-emerald-700 text-white px-2.5 py-1 rounded transition-colors"
                      >
                        RE-ROUTE TO PARTITION
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

      </div>

      {/* =========================================================
          5. CQRS LIVE SIMULATOR & SEPARATED READ MODELS
         ========================================================= */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        
        {/* Write Pipeline (Command Handler Dispatcher) */}
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 h-[360px] flex flex-col justify-between">
          <div>
            <div className="border-b border-slate-900 pb-2.5 mb-3">
              <h3 className="font-display font-semibold text-white text-xs">Write Pipeline: Command Handler</h3>
              <p className="text-[10px] text-slate-400">Dispatch validation commands to append events and trigger messaging</p>
            </div>

            <div className="space-y-2 text-xs">
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-1">
                  <label className="text-[9px] text-slate-500 uppercase font-bold font-sans">Transaction ID</label>
                  <div className="flex bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
                    <input 
                      type="text"
                      value={cmdId}
                      onChange={e => setCmdId(e.target.value)}
                      className="bg-transparent text-[10px] font-mono text-white px-2 py-1.5 flex-1 focus:outline-none min-w-0"
                    />
                    <button onClick={handleRegenId} className="px-1.5 bg-slate-800 text-slate-400 text-[9px] font-bold border-l border-slate-700">
                      GEN
                    </button>
                  </div>
                </div>

                <div className="space-y-1">
                  <label className="text-[9px] text-slate-500 uppercase font-bold font-sans">Amount ($)</label>
                  <input 
                    type="number"
                    value={cmdAmount}
                    onChange={e => setCmdAmount(parseFloat(e.target.value) || 0)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-[10px] font-mono text-white focus:outline-none"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-1">
                  <label className="text-[9px] text-slate-500 uppercase font-bold font-sans">Created By</label>
                  <input 
                    type="text"
                    value={cmdUser}
                    onChange={e => setCmdUser(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-2.5 py-1.5 text-[10px] font-mono text-white focus:outline-none"
                  />
                </div>

                <div className="space-y-1">
                  <label className="text-[9px] text-slate-500 uppercase font-bold font-sans">Enforce Schema</label>
                  <select 
                    value={cmdSchemaVersion}
                    onChange={e => setCmdSchemaVersion(parseInt(e.target.value) || 1)}
                    className="w-full bg-slate-900 border border-slate-800 rounded-lg px-2 py-1.5 text-[10px] text-slate-300 focus:outline-none"
                  >
                    <option value={1}>Created v1 (Standard)</option>
                    <option value={2}>Created v2 (Multi-Tenant)</option>
                  </select>
                </div>
              </div>

              {cmdSchemaVersion === 2 && (
                <div className="space-y-1">
                  <label className="text-[9px] text-amber-500 uppercase font-bold font-sans block">Required Tenant ID (v2 Schema)</label>
                  <input 
                    type="text"
                    value={cmdTenant}
                    onChange={e => setCmdTenant(e.target.value)}
                    className="w-full bg-slate-900 border border-amber-950 text-[10px] font-mono text-amber-400 rounded-lg px-2.5 py-1.5 focus:outline-none"
                  />
                </div>
              )}

              <div className="flex items-center gap-2 pt-1.5">
                <input 
                  type="checkbox"
                  id="fail_handler"
                  checked={simulateFailure}
                  onChange={e => setSimulateFailure(e.target.checked)}
                  className="rounded border-slate-800 bg-slate-900 text-indigo-600 focus:ring-0 focus:ring-offset-0"
                />
                <label htmlFor="fail_handler" className="text-[10px] font-bold text-rose-400 select-none cursor-pointer flex items-center gap-1">
                  Inject Projection Connectivity Outage
                </label>
              </div>
            </div>
          </div>

          <button 
            onClick={handleDispatchCommand}
            disabled={processingState !== "idle"}
            className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:opacity-40 text-white rounded-lg py-2 text-xs font-bold transition-all flex items-center justify-center gap-2 shadow-md"
          >
            {processingState === "idle" ? <Send size={13} /> : <RefreshCw size={13} className="animate-spin" />}
            {processingState === "idle" && "EXECUTE CREATE TRANSACTION COMMAND"}
            {processingState === "validating" && "VALIDATING VIA SCHEMA REGISTRY..."}
            {processingState === "appending" && "APPENDING TO EVENTSTOREDB..."}
            {processingState === "publishing" && "DISPATCHING KAFKA ENVOY..."}
            {processingState === "routing" && "ROUTING TO RETRY ENGINE..."}
            {processingState === "success" && "TRANSACTION PROGRESSED HEALTHY!"}
          </button>
        </div>

        {/* Read Pipeline (Separation Projection Database) */}
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 h-[360px] col-span-2 flex flex-col">
          <div className="border-b border-slate-900 pb-2.5 mb-3 flex items-center justify-between">
            <div>
              <h3 className="font-display font-semibold text-white text-xs">Read Pipeline: Projected Materialized Query Model</h3>
              <p className="text-[10px] text-slate-400">Strictly separated query replica database. Re-evaluated reactively upon message routing.</p>
            </div>
            <span className="text-[9px] font-mono bg-emerald-950/20 text-emerald-400 border border-emerald-900/30 px-2 py-0.5 rounded">
              Sync complete
            </span>
          </div>

          <div className="flex-1 overflow-y-auto min-h-0">
            {readModel.length === 0 ? (
              <div className="h-full flex flex-col items-center justify-center text-slate-600 text-center text-xs gap-1">
                <span>No query projection models compiled on active databases.</span>
              </div>
            ) : (
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr className="border-b border-slate-900 text-slate-500 font-bold font-sans">
                    <th className="pb-2">Transaction ID</th>
                    <th className="pb-2 text-right">Settled Amount</th>
                    <th className="pb-2 text-center">Engine Status</th>
                    <th className="pb-2 text-center">Applied Event Version</th>
                    <th className="pb-2 text-right">Resolved Time</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-900 font-mono">
                  {readModel.map(model => (
                    <tr key={model.id} className="text-slate-300 hover:bg-slate-900/10 transition-colors">
                      <td className="py-2 text-slate-200 font-bold">{model.id}</td>
                      <td className="py-2 text-right text-emerald-400 font-bold">${model.amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}</td>
                      <td className="py-2 text-center">
                        <span className={`text-[9px] font-bold px-2 py-0.5 rounded ${model.status === "APPROVED" ? "bg-emerald-950/20 text-emerald-400 border border-emerald-900/30" : "bg-rose-950/20 text-rose-400 border border-rose-900/30"}`}>
                          {model.status}
                        </span>
                      </td>
                      <td className="py-2 text-center text-indigo-400 font-bold">v{model.version}</td>
                      <td className="py-2 text-right text-slate-500">{new Date(model.updatedAt).toLocaleTimeString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

      </div>
    </div>
  );
}
