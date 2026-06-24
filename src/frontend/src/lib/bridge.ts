/**
 * Thin facade over Go bound methods (via Wails) or mock data when running standalone.
 * All frontend code calls this module — never wailsjs directly.
 */
import type { VMInstance, Session, AuditEntry, CLIStatus } from "@/types/domain";

export const MOCK_VMS: VMInstance[] = [
  { id: 1,  name: "web-prod-01",      provider: "aws",   region: "us-east-1",       size: "t3.large",          state: "running",      privateIP: "10.0.1.12",   iid: "i-0f3a9c12", tags: ["env:prod","team:web","app:storefront"], canConnect: true  },
  { id: 2,  name: "api-prod-02",      provider: "aws",   region: "us-east-1",       size: "m5.xlarge",         state: "running",      privateIP: "10.0.1.45",   iid: "i-0b71de84", tags: ["env:prod","team:platform"],             canConnect: true  },
  { id: 3,  name: "worker-prod-03",   provider: "aws",   region: "us-east-1",       size: "c5.2xlarge",        state: "stopped",      privateIP: "10.0.2.11",   iid: "i-0a4471f0", tags: ["env:prod","team:data"],                 canConnect: true  },
  { id: 4,  name: "db-replica-01",    provider: "aws",   region: "eu-west-1",       size: "r5.large",          state: "running",      privateIP: "10.1.0.9",    iid: "i-09cc2a3e", tags: ["env:prod","team:data","tier:db"],        canConnect: false },
  { id: 5,  name: "bastion-eu",       provider: "aws",   region: "eu-west-1",       size: "t3.micro",          state: "running",      privateIP: "10.1.0.2",    iid: "i-02ee9b17", tags: ["env:shared","role:bastion"],             canConnect: true  },
  { id: 6,  name: "batch-ap-01",      provider: "aws",   region: "ap-southeast-2",  size: "c5.4xlarge",        state: "transitioning",privateIP: "10.2.0.30",   iid: "i-0d56af91", tags: ["env:prod","team:ml"],                   canConnect: true  },
  { id: 18, name: "monitoring-01",    provider: "aws",   region: "us-east-1",       size: "t3.medium",         state: "running",      privateIP: "10.0.3.5",    iid: "i-0c18be72", tags: ["env:shared","role:observability"],       canConnect: true  },
  { id: 7,  name: "web-staging-01",   provider: "azure", region: "eastus",          size: "Standard_D4s_v5",   state: "running",      privateIP: "172.16.0.8",  iid: "vm-7be3a1",  tags: ["env:staging","team:web"],               canConnect: true  },
  { id: 8,  name: "api-staging-02",   provider: "azure", region: "eastus",          size: "Standard_D2s_v5",   state: "stopped",      privateIP: "172.16.0.21", iid: "vm-2c98ff",  tags: ["env:staging","team:platform"],          canConnect: true  },
  { id: 9,  name: "analytics-01",     provider: "azure", region: "westeurope",      size: "Standard_E8s_v5",   state: "running",      privateIP: "172.17.0.5",  iid: "vm-91ad04",  tags: ["env:prod","team:data"],                 canConnect: true  },
  { id: 10, name: "jumphost-we",      provider: "azure", region: "westeurope",      size: "Standard_B2s",      state: "running",      privateIP: "172.17.0.2",  iid: "vm-44ab90",  tags: ["env:shared","role:bastion"],             canConnect: true  },
  { id: 11, name: "ml-train-01",      provider: "azure", region: "westeurope",      size: "Standard_NC6s_v3",  state: "transitioning",privateIP: "172.17.1.40", iid: "vm-0fd7c2",  tags: ["env:prod","team:ml","gpu:v100"],        canConnect: true  },
  { id: 19, name: "backup-we",        provider: "azure", region: "westeurope",      size: "Standard_D2s_v5",   state: "running",      privateIP: "172.17.0.60", iid: "vm-7712ce",  tags: ["env:prod","role:backup"],               canConnect: true  },
  { id: 12, name: "web-dev-01",       provider: "gcp",   region: "us-central1",     size: "e2-standard-4",     state: "running",      privateIP: "10.128.0.12", iid: "instance-3a17", tags: ["env:dev","team:web"],                canConnect: true  },
  { id: 13, name: "api-dev-02",       provider: "gcp",   region: "us-central1",     size: "n2-standard-8",     state: "running",      privateIP: "10.128.0.33", iid: "instance-8c40", tags: ["env:dev","team:platform"],           canConnect: true  },
  { id: 14, name: "data-pipeline-01", provider: "gcp",   region: "us-central1",     size: "n2-standard-16",    state: "stopped",      privateIP: "10.128.1.4",  iid: "instance-1de9", tags: ["env:prod","team:data"],              canConnect: true  },
  { id: 20, name: "ci-runner-03",     provider: "gcp",   region: "us-central1",     size: "e2-standard-8",     state: "running",      privateIP: "10.128.0.77", iid: "instance-77b2", tags: ["env:ci","role:runner"],              canConnect: true  },
  { id: 15, name: "cache-eu-01",      provider: "gcp",   region: "europe-west1",    size: "e2-medium",         state: "running",      privateIP: "10.132.0.7",  iid: "instance-0a7c", tags: ["env:prod","tier:cache"],             canConnect: true  },
  { id: 16, name: "search-eu-02",     provider: "gcp",   region: "europe-west1",    size: "n2-highmem-4",      state: "running",      privateIP: "10.132.0.19", iid: "instance-19fa", tags: ["env:prod","team:search"],            canConnect: false },
  { id: 17, name: "gpu-render-01",    provider: "gcp",   region: "europe-west1",    size: "a2-highgpu-1g",     state: "unknown",      privateIP: "10.132.1.50", iid: "instance-50bb", tags: ["env:prod","team:ml","gpu:a100"],     canConnect: true  },
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
  running:      { label: "Running",     color: "#16A34A" },
  stopped:      { label: "Stopped",     color: "#DC2626" },
  transitioning:{ label: "Transitioning", color: "#D97706" },
  unknown:      { label: "Unknown",     color: "#64748B" },
};

export async function mockLogin(_method: string): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, 1400));
}

export async function mockRefresh(): Promise<{ aws: number; gcp: number; azure: number }> {
  return new Promise(resolve => setTimeout(() => resolve({ aws: 650, gcp: 1050, azure: 1400 }), 0));
}

export async function mockGetCLIs(): Promise<CLIStatus[]> {
  return MOCK_CLI;
}

export async function mockGetAuditLog(): Promise<AuditEntry[]> {
  return MOCK_AUDIT;
}
