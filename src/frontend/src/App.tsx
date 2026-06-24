import { useEffect } from "react";
import { useAppStore } from "@/store/useAppStore";
import { useShallow } from "zustand/react/shallow";

import { Sidebar } from "@/shell/Sidebar";
import { StatusBar } from "@/shell/StatusBar";

import { AuthScreen } from "@/features/auth/AuthScreen";
import { InventoryScreen } from "@/features/inventory/InventoryScreen";
import { SessionsScreen } from "@/features/sessions/SessionsScreen";
import { AuditScreen } from "@/features/audit/AuditScreen";
import { SettingsScreen } from "@/features/settings/SettingsScreen";

import { Toaster } from "@/ui/sonner";
import { ConfirmModal } from "@/ui/ConfirmModal";

function ScreenContent() {
  const screen = useAppStore(s => s.screen);
  switch (screen) {
    case "inventory": return <InventoryScreen />;
    case "sessions":  return <SessionsScreen />;
    case "audit":     return <AuditScreen />;
    case "settings":  return <SettingsScreen />;
  }
}

function MainApp() {
  return (
    <div className="h-screen flex flex-col overflow-hidden"
      style={{ background: "var(--background)", color: "var(--foreground)", fontFamily: "var(--font-sans)", fontSize: 14 }}
    >
      <div className="flex flex-1 min-h-0">
        <Sidebar />
        <main className="flex-1 min-w-0 flex flex-col overflow-hidden">
          <ScreenContent />
        </main>
      </div>
      <StatusBar />
    </div>
  );
}

export default function App() {
  const { loggedIn, theme } = useAppStore(useShallow(s => ({
    loggedIn: s.loggedIn,
    theme: s.theme,
  })));

  useEffect(() => {
    if (theme === "dark") document.documentElement.classList.add("dark");
    else document.documentElement.classList.remove("dark");
  }, [theme]);

  useEffect(() => {
    const id = setInterval(() => useAppStore.getState().incrementTick(), 1000);
    return () => clearInterval(id);
  }, []);

  return (
    <>
      {loggedIn ? <MainApp /> : <AuthScreen />}
      <Toaster />
      <ConfirmModal />
    </>
  );
}
