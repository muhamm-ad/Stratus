export type ProviderID = "aws" | "azure" | "gcp";

export type VMState = "running" | "stopped" | "transitioning" | "unknown";

export interface VMInstance {
  id: string;         // cloud-native instance ID (e.g. i-0f3a9c12, /subscriptions/…, 12345)
  name: string;
  provider: ProviderID;
  region: string;
  state: VMState;
  size: string;       // instance type / machine type (e.g. t3.large, n2-standard-4)
  platform: string;   // "linux" | "windows"
  privateIP: string;
  publicIP: string;
  osUser: string;     // default login user (ec2-user, ubuntu, azureuser, …)
  tags: string[];     // ["key:value", …]
  canConnect: boolean;
}

export interface ConnectCheck {
  canConnect: boolean;
  method: "ssm" | "bastion" | "ssh" | "iap";
  reason?: string;
}

export interface Session {
  sid: string;
  vmId: string;       // matches VMInstance.id
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
