export type ProviderID = "aws" | "azure" | "gcp";

export type VMState = "running" | "stopped" | "transitioning" | "unknown";

export interface VMInstance {
  id: number;
  name: string;
  provider: ProviderID;
  region: string;
  state: VMState;
  size: string;
  privateIP: string;
  iid: string;
  tags: string[];
  canConnect: boolean;
}

export interface ConnectCheck {
  canConnect: boolean;
  method: "ssm" | "bastion" | "ssh" | "iap";
  reason?: string;
}

export interface Session {
  sid: string;
  vmId: number;
  openedAt: number;
}

export interface AuditEntry {
  timestamp: string;
  user: string;
  vmName: string;
  provider: ProviderID;
  method: string;
  result: "success" | "failure";
}

export interface Provider {
  id: ProviderID;
  connected: boolean;
  identity?: string;
}

export interface CLIStatus {
  name: string;
  installed: boolean;
}

export type SSOStatus = "connected" | "disconnected" | "connecting";

export type Screen = "inventory" | "sessions" | "audit" | "settings";

export type ViewMode = "table" | "cards" | "grouped";

export interface ToastItem {
  id: string;
  kind: "success" | "error" | "warning";
  title: string;
  body?: string;
  action?: { label: string; fn: () => void };
}

export interface ModalItem {
  title: string;
  body: string;
  cancelLabel: string;
  confirmLabel: string;
  danger?: boolean;
  onConfirm: () => void;
}
