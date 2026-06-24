import { create } from "zustand";
import type {
  Screen, ViewMode, VMInstance, Session, ToastItem, ModalItem, ProviderID, SSOStatus,
	VMState,
} from "@/types/domain";


import { MOCK_VMS, PROVIDER_META } from "@/lib/bridge";

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
  selectedVmId: number | null;
  selectedVmIds: number[];
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
  setTheme: (t: "dark" | "light" | "system") => void;
  toggleTheme: () => void;
  go: (screen: Screen) => void;
  login: (method: string) => Promise<void>;
  signOut: () => void;
  refresh: () => void;
  reconnect: (p: ProviderID) => void;
  ssoConnect: (p: ProviderID) => void;
  ssoDisconnect: (p: ProviderID) => void;
  toggleProvider: (p: ProviderID) => void;
  openVm: (id: number) => void;
  closeDrawer: () => void;
  connect: (vm: VMInstance) => void;
  connectById: (id: number) => void;
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
  toggleSelectVm: (id: number) => void;
  selectAllVms: (ids: number[]) => void;
  clearSelection: () => void;
  bulkConnect: () => void;
  setSessionSearch: (q: string) => void;
  setAuditSearch: (q: string) => void;
}

type Store = AppState & AppActions;

let _timers: ReturnType<typeof setTimeout>[] = [];
function later(fn: () => void, ms: number) {
  const t = setTimeout(fn, ms);
  _timers.push(t);
  return t;
}




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

  sso: { aws: "connected", azure: "connected", gcp: "connected" },
  providerLoading: { aws: false, azure: false, gcp: false },
  providerError: { aws: false, azure: true, gcp: false },

  vms: MOCK_VMS,
  selectedVmId: null,
  selectedVmIds: [],
  sessions: [
    { sid: "s1", vmId: 1, openedAt: Date.now() - 8 * 60000 },
    { sid: "s2", vmId: 12, openedAt: Date.now() - 23 * 60000 },
  ],
  toasts: [],
  modal: null,
  autoRefresh: false,
  tickCount: 0,
  density: "comfortable",
  sessionSearch: "",
  auditSearch: "",

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

  login: async (method) => {
    set({ loginStage: method });
    await new Promise<void>(r => later(r, 1400));
    set({ loggedIn: true, loginStage: null, screen: "inventory" });
    get().refresh();
  },

  signOut: () => {
    set({
      modal: {
        title: "Sign out of Stratus?",
        body: "You will need to re-authenticate with each provider to access your inventory again.",
        cancelLabel: "Stay",
        confirmLabel: "Sign out",
        danger: true,
        onConfirm: () => set({ loggedIn: false, modal: null }),
      },
    });
  },

  refresh: () => {
    const { providerError } = get();
    set({
      providerLoading: {
        aws: true,
        azure: providerError.azure ? false : true,
        gcp: true,
      },
    });
    later(() => set(s => ({ providerLoading: { ...s.providerLoading, aws: false } })), 650);
    later(() => set(s => ({ providerLoading: { ...s.providerLoading, gcp: false } })), 1050);
    later(() => set(s => ({ providerLoading: { ...s.providerLoading, azure: false } })), 1400);
  },

  reconnect: (p) => {
    set(s => ({ providerError: { ...s.providerError, [p]: false }, providerLoading: { ...s.providerLoading, [p]: true } }));
    later(() => {
      set(s => ({ providerLoading: { ...s.providerLoading, [p]: false } }));
      get().toast("success", `${PROVIDER_META[p].label} reconnected`, PROVIDER_META[p].identity);
    }, 1100);
  },

  ssoConnect: (p) => {
    set(s => ({ sso: { ...s.sso, [p]: "connecting" } }));
    later(() => {
      set(s => ({
        sso: { ...s.sso, [p]: "connected" },
        providerError: { ...s.providerError, [p]: false },
        providerLoading: { ...s.providerLoading, [p]: true },
      }));
      later(() => {
        set(s => ({ providerLoading: { ...s.providerLoading, [p]: false } }));
        get().toast("success", `${PROVIDER_META[p].label} connected`, PROVIDER_META[p].identity);
      }, 800);
    }, 1400);
  },

  ssoDisconnect: (p) => {
    set(s => ({ sso: { ...s.sso, [p]: "disconnected" }, providerError: { ...s.providerError, [p]: false } }));
  },

  toggleProvider: (p) => {
    const { sso } = get();
    if (sso[p] === "connected") get().ssoDisconnect(p);
    else if (sso[p] !== "connecting") get().ssoConnect(p);
  },

  openVm: (id) => set({ selectedVmId: id }),
  closeDrawer: () => set({ selectedVmId: null }),

  connect: (vm) => {
    if (!vm.canConnect) return;
    set(s => {
      const exists = s.sessions.some(x => x.vmId === vm.id);
      const sessions = exists ? s.sessions : [...s.sessions, { sid: `sx${Date.now()}`, vmId: vm.id, openedAt: Date.now() }];
      return { sessions, selectedVmId: null };
    });
    get().toast("success", "Session opened", `${vm.name} · ${PROVIDER_META[vm.provider].method}`);
  },

  connectById: (id) => {
    const vm = get().vms.find(v => v.id === id);
    if (vm) get().connect(vm);
  },

  closeSession: (sid) => set(s => ({ sessions: s.sessions.filter(x => x.sid !== sid) })),

  toast: (kind, title, body?, action?) => {
    const id = `to${Date.now()}${Math.random()}`;
    set(s => ({ toasts: [...s.toasts, { id, kind, title, body, action }] }));
    later(() => get().dismissToast(id), 4200);
  },

  dismissToast: (id) => set(s => ({ toasts: s.toasts.filter(t => t.id !== id) })),
  openModal: (m) => set({ modal: m }),
  closeModal: () => set({ modal: null }),

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
