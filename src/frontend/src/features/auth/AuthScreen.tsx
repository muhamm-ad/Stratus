import { Loader2, KeyRound } from "lucide-react";
import { useAppStore } from "@/store/useAppStore";
import { StratusMark } from "@/ui/StratusLogo";

const LOGIN_METHODS = [
  { key: "okta",  label: "Okta SSO"  },
  { key: "saml",  label: "SAML 2.0"  },
];

export function AuthScreen() {
  const loginStage = useAppStore(s => s.loginStage);
  const login = useAppStore(s => s.login);
  const busy = loginStage !== null;

  return (
    <div
      className="h-screen flex items-center justify-center px-6"
      style={{ background: "var(--background)", color: "var(--foreground)", fontFamily: "var(--font-sans)" }}
    >
      <div className="w-[400px] max-w-full animate-fade-up">

        {/* Header */}
        <div className="flex flex-col items-center text-center mb-[30px]">
          <div
            className="w-12 h-12 rounded-[13px] flex items-center justify-center"
          >
            <StratusMark size={80} />
          </div>
          <div className="text-[22px] font-bold tracking-tight text-foreground">Stratus</div>
          <div className="text-[13px] font-medium text-muted-foreground mt-[3px]">Multi-cloud VM gateway</div>
        </div>

        {/* Card */}
        <div className="border rounded-[12px] p-6" style={{ borderColor: "var(--border)", background: "var(--surface)" }}>
          <div className="flex items-center gap-3 mb-[18px]">
            <span className="flex-1 h-px" style={{ background: "var(--border)" }} />
            <span className="text-[11px] font-semibold tracking-[.14em] text-muted-foreground">ENTERPRISE LOGIN</span>
            <span className="flex-1 h-px" style={{ background: "var(--border)" }} />
          </div>

          <div className="flex gap-2.5">
            {LOGIN_METHODS.map(m => {
              const loading = loginStage === m.key;
              return (
                <button
                  key={m.key}
                  onClick={() => login(m.key)}
                  disabled={busy}
                  className="flex-1 inline-flex items-center justify-center gap-2 h-[46px] rounded-[8px] border font-semibold text-[13px] cursor-pointer transition-all duration-150 disabled:opacity-55 disabled:cursor-default"
                  style={{
                    borderColor: "var(--border)",
                    background: loading ? "var(--primary)" : "var(--raised)",
                    color: loading ? "#fff" : "var(--foreground)",
                  }}
                >
                  {loading
                    ? <><Loader2 size={16} className="animate-spin" /> Redirecting…</>
                    : <><KeyRound size={17} /> {m.label}</>
                  }
                </button>
              );
            })}
          </div>

          <div className="text-center mt-[18px] text-[12.5px] text-muted-foreground">
            Need access to a different tenant?{" "}
            <a
              href="#"
              onClick={e => e.preventDefault()}
              className="font-semibold no-underline"
              style={{ color: "var(--primary)" }}
            >
              Contact Platform Ops
            </a>
          </div>
        </div>

        <div className="text-center mt-[18px] text-[12px] text-muted-foreground leading-relaxed">
          Sign in once with your identity provider.<br />
          Connect AWS, Azure &amp; GCP from inside the app.
        </div>
      </div>
    </div>
  );
}
