import { create } from "zustand";
import type {
  Screen, ViewMode, VMInstance, Session, ToastItem, ModalItem, ProviderID, SSOStatus,
  VMState,
} from "@/types/domain";

import { PROVIDER_META } from "@/lib/bridge";
import * as bridge from "@/lib/bridge";

export type FilterProvider = "all" | ProviderID;
export type FilterState = "all" | VMState;
export type Density = "comfortable" | "compact";

interface AppState {
  theme: "dark" | "light";
  prefTheme: "dark" | "light" | "system";
  screen: Screen;
  loggedIn: boolean;
  loginStage: string | null;

  view: ViewMode;
  search: string;
  filterProvider: FilterProvider;
  filterState: FilterState;
  filterRegion: string;
  filterTag: string;

  sso: Record<ProviderID, SSOStatus>;
  providerLoading: Record<ProviderID, boolean>;
  providerError: Record<ProviderID, boolean>;

  vms: VMInstance[];
  selectedVmId: string | null;
  selectedVmIds: string[];
  sessions: Session[];
  toasts: ToastItem[];
  modal: ModalItem | null;
  autoRefresh: boolean;
  tickCount: number;
  density: Density;
  sessionSearch: string;
  auditSearch: string;
}

interface AppActions {
  init: () => Promise<void>;
  setTheme: (t: "dark" | "light" | "system") => void;
  toggleTheme: () => void;
  go: (screen: Screen) => void;
  login: (method: string) => Promise<void>;
  signOut: () => void;
  refresh: () => Promise<void>;
  reconnect: (p: ProviderID) => Promise<void>;
  ssoConnect: (p: ProviderID) => Promise<void>;
  ssoDisconnect: (p: ProviderID) => void;
  toggleProvider: (p: ProviderID) => void;
  openVm: (id: string) => void;
  closeDrawer: () => void;
  connect: (vm: VMInstance) => void;
  connectById: (id: string) => void;
  closeSession: (sid: string) => void;
  toast: (kind: ToastItem["kind"], title: string, body?: string, action?: ToastItem["action"]) => void;
  dismissToast: (id: string) => void;
  openModal: (m: ModalItem) => void;
  closeModal: () => void;
  setSearch: (q: string) => void;
  setFilterProvider: (p: FilterProvider) => void;
  setFilterState: (s: FilterState) => void;
  setFilterRegion: (r: string) => void;
  setFilterTag: (t: string) => void;
  setView: (v: ViewMode) => void;
  clearFilters: () => void;
  toggleAutoRefresh: () => void;
  incrementTick: () => void;
  setDensity: (d: Density) => void;
  toggleSelectVm: (id: string) => void;
  selectAllVms: (ids: string[]) => void;
  clearSelection: () => void;
  bulkConnect: () => void;
  setSessionSearch: (q: string) => void;
  setAuditSearch: (q: string) => void;
}

type Store = AppState & AppActions;

const PROVIDERS: ProviderID[] = ["aws", "azure", "gcp"];

export const useAppStore = create<Store>((set, get) => ({
  theme: "dark",
  prefTheme: "dark",
  screen: "inventory",
  loggedIn: false,
  loginStage: null,

  view: "table",
  search: "",
  filterProvider: "all",
  filterState: "all",
  filterRegion: "all",
  filterTag: "",

  sso: { aws: "disconnected", azure: "disconnected", gcp: "disconnected" },
  providerLoading: { aws: false, azure: false, gcp: false },
  providerError: { aws: false, azure: false, gcp: false },

  vms: [],
  selectedVmId: null,
  selectedVmIds: [],
  sessions: [],
  toasts: [],
  modal: null,
  autoRefresh: false,
  tickCount: 0,
  density: "comfortable",
  sessionSearch: "",
  auditSearch: "",

  // ── Startup ──────────────────────────────────────────────────────────────

  init: async () => {
    try {
      const authed = await bridge.isAuthenticated();
      if (authed) {
        set({ loggedIn: true, screen: "inventory" });
        await get().refresh();
      }
    } catch {
      // Backend unreachable (standalone browser dev) — stay on auth screen.
    }
  },

  // ── Theme ─────────────────────────────────────────────────────────────────

  setTheme: (t) => {
    const resolved = t === "system" ? "dark" : t;
    set({ prefTheme: t === "system" ? "system" : resolved, theme: resolved });
    if (resolved === "dark") {
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.classList.remove("dark");
    }
  },

  toggleTheme: () => get().setTheme(get().theme === "dark" ? "light" : "dark"),

  go: (screen) => set({ screen, selectedVmId: null, selectedVmIds: [] }),

  // ── Auth ──────────────────────────────────────────────────────────────────

  login: async (method) => {
    set({ loginStage: "browser" });
    try {
      await bridge.login(method);
      set({ loggedIn: true, loginStage: null, screen: "inventory" });
      // Auto-connect and list all providers after Entra login.
      await get().refresh();
    } catch (e) {
      set({ loginStage: null });
      get().toast("error", "Login failed", String(e));
    }
  },

  signOut: () => {
    set({
      modal: {
        title: "Sign out of Stratus?",
        body: "You will need to re-authenticate with each provider to access your inventory again.",
        cancelLabel: "Stay",
        confirmLabel: "Sign out",
        danger: true,
        onConfirm: async () => {
          await bridge.logout();
          set({
            loggedIn: false,
            modal: null,
            vms: [],
            sso: { aws: "disconnected", azure: "disconnected", gcp: "disconnected" },
            providerError: { aws: false, azure: false, gcp: false },
            sessions: [],
          });
        },
      },
    });
  },

  // ── Provider connection & inventory fetch ─────────────────────────────────

  refresh: async () => {
    // Connect + list every provider whose SSO toggle is on, in parallel.
    await Promise.all(
      PROVIDERS.map(async (p) => {
        if (get().sso[p] === "disconnected") return;
        set(s => ({ providerLoading: { ...s.providerLoading, [p]: true }, providerError: { ...s.providerError, [p]: false } }));
        try {
          // If not yet connected, do the silent exchange first.
          if (get().sso[p] !== "connected") {
            await bridge.connectProvider(p);
            set(s => ({ sso: { ...s.sso, [p]: "connected" } }));
          }
          const vms = await bridge.listInstances(p, "");
          set(s => ({
            vms: [...s.vms.filter(v => v.provider !== p), ...vms],
            providerLoading: { ...s.providerLoading, [p]: false },
          }));
        } catch (e) {
          set(s => ({
            providerLoading: { ...s.providerLoading, [p]: false },
            providerError: { ...s.providerError, [p]: true },
          }));
          get().toast("error", `${PROVIDER_META[p].label} failed to load`, String(e));
        }
      })
    );
  },

  reconnect: async (p) => {
    set(s => ({ providerError: { ...s.providerError, [p]: false }, providerLoading: { ...s.providerLoading, [p]: true } }));
    try {
      await bridge.connectProvider(p);
      set(s => ({ sso: { ...s.sso, [p]: "connected" } }));
      const vms = await bridge.listInstances(p, "");
      set(s => ({
        vms: [...s.vms.filter(v => v.provider !== p), ...vms],
        providerLoading: { ...s.providerLoading, [p]: false },
      }));
      get().toast("success", `${PROVIDER_META[p].label} reconnected`, PROVIDER_META[p].identity);
    } catch (e) {
      set(s => ({
        providerLoading: { ...s.providerLoading, [p]: false },
        providerError: { ...s.providerError, [p]: true },
      }));
      get().toast("error", `${PROVIDER_META[p].label} reconnect failed`, String(e));
    }
  },

  ssoConnect: async (p) => {
    set(s => ({ sso: { ...s.sso, [p]: "connecting" } }));
    try {
      await bridge.connectProvider(p);
      set(s => ({ sso: { ...s.sso, [p]: "connected" }, providerLoading: { ...s.providerLoading, [p]: true } }));
      const vms = await bridge.listInstances(p, "");
      set(s => ({
        vms: [...s.vms.filter(v => v.provider !== p), ...vms],
        providerLoading: { ...s.providerLoading, [p]: false },
      }));
      get().toast("success", `${PROVIDER_META[p].label} connected`, PROVIDER_META[p].identity);
    } catch (e) {
      set(s => ({ sso: { ...s.sso, [p]: "disconnected" }, providerLoading: { ...s.providerLoading, [p]: false }, providerError: { ...s.providerError, [p]: true } }));
      get().toast("error", `${PROVIDER_META[p].label} connect failed`, String(e));
    }
  },

  ssoDisconnect: (p) => {
    set(s => ({
      sso: { ...s.sso, [p]: "disconnected" },
      providerError: { ...s.providerError, [p]: false },
      vms: s.vms.filter(v => v.provider !== p),
    }));
  },

  toggleProvider: (p) => {
    const { sso } = get();
    if (sso[p] === "connected") get().ssoDisconnect(p);
    else if (sso[p] !== "connecting") void get().ssoConnect(p);
  },

  // ── VM selection & sessions ───────────────────────────────────────────────

  openVm: (id) => set({ selectedVmId: id }),
  closeDrawer: () => set({ selectedVmId: null }),

  connect: (vm) => {
    if (!vm.canConnect) return;
    set(s => {
      const exists = s.sessions.some(x => x.vmId === vm.id);
      const sessions = exists
        ? s.sessions
        : [...s.sessions, { sid: `sx${Date.now()}`, vmId: vm.id, openedAt: Date.now() }];
      return { sessions, selectedVmId: null };
    });
    get().toast("success", "Session opened", `${vm.name} · ${PROVIDER_META[vm.provider].method}`);
  },

  connectById: (id) => {
    const vm = get().vms.find(v => v.id === id);
    if (vm) get().connect(vm);
  },

  closeSession: (sid) => set(s => ({ sessions: s.sessions.filter(x => x.sid !== sid) })),

  // ── Toasts & modals ───────────────────────────────────────────────────────

  toast: (kind, title, body?, action?) => {
    const id = `to${Date.now()}${Math.random()}`;
    set(s => ({ toasts: [...s.toasts, { id, kind, title, body, action }] }));
    setTimeout(() => get().dismissToast(id), 4200);
  },

  dismissToast: (id) => set(s => ({ toasts: s.toasts.filter(t => t.id !== id) })),
  openModal: (m) => set({ modal: m }),
  closeModal: () => set({ modal: null }),

  // ── Filters ───────────────────────────────────────────────────────────────

  setSearch: (q) => set({ search: q }),
  setFilterProvider: (p) => set({ filterProvider: p }),
  setFilterState: (s) => set({ filterState: s }),
  setFilterRegion: (r) => set({ filterRegion: r }),
  setFilterTag: (t) => set({ filterTag: t }),
  setView: (v) => set({ view: v }),
  clearFilters: () => set({ filterProvider: "all", filterState: "all", filterRegion: "all", search: "", filterTag: "" }),
  toggleAutoRefresh: () => set(s => ({ autoRefresh: !s.autoRefresh })),
  incrementTick: () => set(s => ({ tickCount: s.tickCount + 1 })),
  setDensity: (d) => set({ density: d }),

  // ── Bulk selection ────────────────────────────────────────────────────────

  toggleSelectVm: (id) => set(s => ({
    selectedVmIds: s.selectedVmIds.includes(id)
      ? s.selectedVmIds.filter(x => x !== id)
      : [...s.selectedVmIds, id],
  })),
  selectAllVms: (ids) => set({ selectedVmIds: ids }),
  clearSelection: () => set({ selectedVmIds: [] }),

  bulkConnect: () => {
    const { selectedVmIds, vms } = get();
    let opened = 0;
    for (const id of selectedVmIds) {
      const vm = vms.find(v => v.id === id);
      if (vm?.canConnect) { get().connect(vm); opened++; }
    }
    get().clearSelection();
    if (opened > 1) get().toast("success", `${opened} sessions opened`);
  },

  setSessionSearch: (q) => set({ sessionSearch: q }),
  setAuditSearch: (q) => set({ auditSearch: q }),
}));
