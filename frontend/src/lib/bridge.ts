/**
 * Thin facade over Go bound methods (via Wails) with mock fallback when running
 * standalone in a browser (e.g. `npm run dev` without the desktop shell).
 * All frontend code calls this module — never wailsjs directly.
 */
import * as App from "../../wailsjs/go/app/App";
import type { VMInstance, VMState, AuditEntry, CLIStatus, ProviderID } from "@/types/domain";

// True when running inside the Wails desktop shell.
const inWails = typeof window !== "undefined" &&
  (window as unknown as { go?: unknown }).go != null;

// ── Mock data (used as fallback in standalone browser dev) ──────────────────

export const MOCK_VMS: VMInstance[] = [
  { id: "i-0f3a9c12", name: "web-prod-01",      provider: "aws",   region: "us-east-1",      size: "t3.large",         state: "running",       platform: "linux",   privateIP: "10.0.1.12",   publicIP: "",           osUser: "ec2-user",   tags: ["env:prod","team:web","app:storefront"], canConnect: true  },
  { id: "i-0b71de84", name: "api-prod-02",      provider: "aws",   region: "us-east-1",      size: "m5.xlarge",        state: "running",       platform: "linux",   privateIP: "10.0.1.45",   publicIP: "",           osUser: "ec2-user",   tags: ["env:prod","team:platform"],             canConnect: true  },
  { id: "i-0a4471f0", name: "worker-prod-03",   provider: "aws",   region: "us-east-1",      size: "c5.2xlarge",       state: "stopped",       platform: "linux",   privateIP: "10.0.2.11",   publicIP: "",           osUser: "ec2-user",   tags: ["env:prod","team:data"],                 canConnect: false },
  { id: "i-09cc2a3e", name: "db-replica-01",    provider: "aws",   region: "eu-west-1",      size: "r5.large",         state: "running",       platform: "linux",   privateIP: "10.1.0.9",    publicIP: "",           osUser: "ec2-user",   tags: ["env:prod","team:data","tier:db"],        canConnect: false },
  { id: "i-02ee9b17", name: "bastion-eu",       provider: "aws",   region: "eu-west-1",      size: "t3.micro",         state: "running",       platform: "linux",   privateIP: "10.1.0.2",    publicIP: "52.0.1.5",   osUser: "ec2-user",   tags: ["env:shared","role:bastion"],             canConnect: true  },
  { id: "i-0d56af91", name: "batch-ap-01",      provider: "aws",   region: "ap-southeast-2", size: "c5.4xlarge",       state: "transitioning", platform: "linux",   privateIP: "10.2.0.30",   publicIP: "",           osUser: "ec2-user",   tags: ["env:prod","team:ml"],                   canConnect: true  },
  { id: "i-0c18be72", name: "monitoring-01",    provider: "aws",   region: "us-east-1",      size: "t3.medium",        state: "running",       platform: "linux",   privateIP: "10.0.3.5",    publicIP: "",           osUser: "ec2-user",   tags: ["env:shared","role:observability"],       canConnect: true  },
  { id: "vm-7be3a1",  name: "web-staging-01",   provider: "azure", region: "eastus",         size: "Standard_D4s_v5",  state: "running",       platform: "linux",   privateIP: "172.16.0.8",  publicIP: "",           osUser: "azureuser",  tags: ["env:staging","team:web"],               canConnect: true  },
  { id: "vm-2c98ff",  name: "api-staging-02",   provider: "azure", region: "eastus",         size: "Standard_D2s_v5",  state: "stopped",       platform: "linux",   privateIP: "172.16.0.21", publicIP: "",           osUser: "azureuser",  tags: ["env:staging","team:platform"],          canConnect: false },
  { id: "vm-91ad04",  name: "analytics-01",     provider: "azure", region: "westeurope",     size: "Standard_E8s_v5",  state: "running",       platform: "linux",   privateIP: "172.17.0.5",  publicIP: "",           osUser: "azureuser",  tags: ["env:prod","team:data"],                 canConnect: true  },
  { id: "vm-44ab90",  name: "jumphost-we",      provider: "azure", region: "westeurope",     size: "Standard_B2s",     state: "running",       platform: "linux",   privateIP: "172.17.0.2",  publicIP: "",           osUser: "azureuser",  tags: ["env:shared","role:bastion"],             canConnect: true  },
  { id: "vm-0fd7c2",  name: "ml-train-01",      provider: "azure", region: "westeurope",     size: "Standard_NC6s_v3", state: "transitioning", platform: "linux",   privateIP: "172.17.1.40", publicIP: "",           osUser: "azureuser",  tags: ["env:prod","team:ml","gpu:v100"],        canConnect: true  },
  { id: "vm-7712ce",  name: "backup-we",        provider: "azure", region: "westeurope",     size: "Standard_D2s_v5",  state: "running",       platform: "linux",   privateIP: "172.17.0.60", publicIP: "",           osUser: "azureuser",  tags: ["env:prod","role:backup"],               canConnect: true  },
  { id: "instance-3a17", name: "web-dev-01",    provider: "gcp",   region: "us-central1",    size: "e2-standard-4",    state: "running",       platform: "linux",   privateIP: "10.128.0.12", publicIP: "",           osUser: "ubuntu",     tags: ["env:dev","team:web"],                   canConnect: true  },
  { id: "instance-8c40", name: "api-dev-02",    provider: "gcp",   region: "us-central1",    size: "n2-standard-8",    state: "running",       platform: "linux",   privateIP: "10.128.0.33", publicIP: "",           osUser: "ubuntu",     tags: ["env:dev","team:platform"],              canConnect: true  },
  { id: "instance-1de9", name: "data-pipeline-01", provider: "gcp", region: "us-central1",  size: "n2-standard-16",   state: "stopped",       platform: "linux",   privateIP: "10.128.1.4",  publicIP: "",           osUser: "ubuntu",     tags: ["env:prod","team:data"],                 canConnect: false },
  { id: "instance-77b2", name: "ci-runner-03",  provider: "gcp",   region: "us-central1",    size: "e2-standard-8",    state: "running",       platform: "linux",   privateIP: "10.128.0.77", publicIP: "",           osUser: "ubuntu",     tags: ["env:ci","role:runner"],                 canConnect: true  },
  { id: "instance-0a7c", name: "cache-eu-01",   provider: "gcp",   region: "europe-west1",   size: "e2-medium",        state: "running",       platform: "linux",   privateIP: "10.132.0.7",  publicIP: "",           osUser: "ubuntu",     tags: ["env:prod","tier:cache"],                canConnect: true  },
  { id: "instance-19fa", name: "search-eu-02",  provider: "gcp",   region: "europe-west1",   size: "n2-highmem-4",     state: "running",       platform: "linux",   privateIP: "10.132.0.19", publicIP: "",           osUser: "ubuntu",     tags: ["env:prod","team:search"],               canConnect: false },
  { id: "instance-50bb", name: "gpu-render-01", provider: "gcp",   region: "europe-west1",   size: "a2-highgpu-1g",    state: "unknown",       platform: "linux",   privateIP: "10.132.1.50", publicIP: "",           osUser: "ubuntu",     tags: ["env:prod","team:ml","gpu:a100"],        canConnect: true  },
];

export const MOCK_AUDIT: AuditEntry[] = [
  { timestamp: "2026-06-22 09:41:08", user: "dana.k",  vmName: "web-prod-01",     provider: "aws",   method: "SSM",     result: "success" },
  { timestamp: "2026-06-22 09:12:55", user: "dana.k",  vmName: "web-dev-01",      provider: "gcp",   method: "IAP",     result: "success" },
  { timestamp: "2026-06-21 18:30:12", user: "omar.r",  vmName: "analytics-01",    provider: "azure", method: "Bastion", result: "success" },
  { timestamp: "2026-06-21 16:04:39", user: "dana.k",  vmName: "db-replica-01",   provider: "aws",   method: "SSM",     result: "failure" },
  { timestamp: "2026-06-21 14:22:01", user: "lin.t",   vmName: "api-prod-02",     provider: "aws",   method: "SSM",     result: "success" },
  { timestamp: "2026-06-21 11:48:17", user: "omar.r",  vmName: "jumphost-we",     provider: "azure", method: "Bastion", result: "success" },
  { timestamp: "2026-06-20 22:09:50", user: "dana.k",  vmName: "search-eu-02",    provider: "gcp",   method: "IAP",     result: "failure" },
  { timestamp: "2026-06-20 15:31:44", user: "lin.t",   vmName: "ci-runner-03",    provider: "gcp",   method: "IAP",     result: "success" },
  { timestamp: "2026-06-20 10:17:23", user: "dana.k",  vmName: "bastion-eu",      provider: "aws",   method: "SSM",     result: "success" },
];

export const MOCK_CLI: CLIStatus[] = [
  { name: "aws-cli",                installed: true  },
  { name: "session-manager-plugin", installed: false },
  { name: "az-cli",                 installed: true  },
  { name: "gcloud",                 installed: true  },
];

export const PROVIDER_META: Record<string, { label: string; color: string; method: string; identity: string }> = {
  aws:   { label: "AWS",   color: "#FF9900", method: "SSM",     identity: "123456789012 · prod"    },
  azure: { label: "Azure", color: "#0078D4", method: "Bastion", identity: "sub: Stratus-Prod"       },
  gcp:   { label: "GCP",   color: "#4285F4", method: "IAP",     identity: "project: stratus-dev"    },
};

export const STATE_META: Record<string, { label: string; color: string }> = {
  running:      { label: "Running",       color: "#16A34A" },
  stopped:      { label: "Stopped",       color: "#DC2626" },
  transitioning:{ label: "Transitioning", color: "#D97706" },
  unknown:      { label: "Unknown",       color: "#64748B" },
};

// ── Real API calls (Wails) with mock fallback ───────────────────────────────

/** Trigger browser login. Pass an IdP id (e.g. "entra") or omit for the single-provider case. */
export async function login(identityProviderID?: string): Promise<void> {
  if (!inWails) {
    await new Promise<void>(r => setTimeout(r, 1400));
    return;
  }
  if (identityProviderID) {
    await App.LoginWith(identityProviderID);
    return;
  }
  await App.Login();
}

/** Returns true if there is a valid Entra session (token not expired). */
export async function isAuthenticated(): Promise<boolean> {
  if (!inWails) return false;
  return App.IsAuthenticated();
}

/** Clear the Entra session and all provider credentials. */
export async function logout(): Promise<void> {
  if (!inWails) return;
  await App.Logout();
}

/** Silent credential exchange for one cloud provider — no browser. */
export async function connectProvider(cloudProviderID: string): Promise<void> {
  if (!inWails) {
    await new Promise<void>(r => setTimeout(r, 800));
    return;
  }
  await App.Connect(cloudProviderID);
}

/**
 * List VMs for a cloud provider. Returns real data from the cloud when running inside
 * Wails; falls back to the per-provider slice of MOCK_VMS in standalone mode.
 */
export async function listInstances(cloudProviderID: string, accountID: string): Promise<VMInstance[]> {
  if (!inWails) {
    await new Promise<void>(r => setTimeout(r, 600 + Math.random() * 600));
    return MOCK_VMS.filter(v => v.provider === (cloudProviderID as ProviderID));
  }
  const data = await App.ListInstances(cloudProviderID, accountID);
  // Wails generates provider/state as plain string; narrow to the union types
  // we control on the Go side.
  return data.map(d => ({
    ...d,
    provider: d.provider as ProviderID,
    state: d.state as VMState,
  }));
}

export async function mockGetCLIs(): Promise<CLIStatus[]> {
  return MOCK_CLI;
}

export async function mockGetAuditLog(): Promise<AuditEntry[]> {
  return MOCK_AUDIT;
}
