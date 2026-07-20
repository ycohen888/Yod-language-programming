import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { Icon } from "./Icon";
import { xtermThemeForName } from "../lib/terminalTheme";
import type { ShellKind, ThemeName } from "../types";

type Props = {
  /** האם פאנל הטרמינל הוא הפעיל כרגע (לצורך fit/מיקוד) */
  visible: boolean;
  theme: ThemeName;
  fontFamily: string;
  fontSize: number;
  /** תיקיית ברירת מחדל לפתיחת סשן (בדרך כלל שורש הפרויקט) */
  defaultCwd?: string;
};

export type TerminalPanelHandle = {
  newSession: (shell: ShellKind) => string;
  openAndRun: (opts: { exe: string; args: string[]; cwd?: string }) => Promise<void>;
  interruptActive: () => void;
  focus: () => void;
};

type TermSession = { id: string; shell: ShellKind; title: string; cwd?: string };

type Host = {
  term: Terminal;
  fit: FitAddon;
  shell: ShellKind;
  input: string;
  ready: boolean;
};

function cdPrefix(shell: ShellKind, cwd: string): string {
  return shell === "cmd" ? `cd /d "${cwd}" & ` : `Set-Location -LiteralPath "${cwd}"; `;
}

function buildInvoke(shell: ShellKind, exe: string, args: string[]): string {
  const quoted = args.map((a) => `"${a}"`).join(" ");
  return shell === "cmd" ? `"${exe}" ${quoted}` : `& "${exe}" ${quoted}`;
}

export const TerminalPanel = forwardRef<TerminalPanelHandle, Props>(function TerminalPanel(
  { visible, theme, fontFamily, fontSize, defaultCwd },
  ref
) {
  const [sessions, setSessions] = useState<TermSession[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [shellMenu, setShellMenu] = useState<{ x: number; y: number } | null>(null);

  const hostsRef = useRef<Map<string, Host>>(new Map());
  const elsRef = useRef<Map<string, HTMLDivElement>>(new Map());
  const wrapRef = useRef<HTMLDivElement>(null);
  const counterRef = useRef(0);
  const activeIdRef = useRef<string | null>(null);
  const defaultCwdRef = useRef<string | undefined>(defaultCwd);
  const themeRef = useRef<ThemeName>(theme);
  const fontRef = useRef({ fontFamily, fontSize });

  activeIdRef.current = activeId;
  defaultCwdRef.current = defaultCwd;
  themeRef.current = theme;
  fontRef.current = { fontFamily, fontSize };

  const handleInput = useCallback((id: string, data: string) => {
    const host = hostsRef.current.get(id);
    if (!host) return;
    for (const ch of data) {
      if (ch === "\r" || ch === "\n") {
        host.term.write("\r\n");
        void window.yod.terminal.write(id, host.input + "\r\n");
        host.input = "";
      } else if (ch === "\u007f" || ch === "\b") {
        if (host.input.length > 0) {
          host.input = host.input.slice(0, -1);
          host.term.write("\b \b");
        }
      } else if (ch === "\u0003") {
        host.term.write("^C\r\n");
        void window.yod.terminal.write(id, "\x03");
        host.input = "";
      } else if (ch >= " " || ch === "\t") {
        host.input += ch;
        host.term.write(ch);
      }
    }
  }, []);

  const createHost = useCallback(
    async (session: TermSession, el: HTMLDivElement) => {
      const term = new Terminal({
        fontFamily: fontRef.current.fontFamily,
        fontSize: fontRef.current.fontSize,
        theme: xtermThemeForName(themeRef.current),
        cursorBlink: true,
        convertEol: true,
        scrollback: 5000,
      });
      const fit = new FitAddon();
      term.loadAddon(fit);
      term.open(el);
      try {
        fit.fit();
      } catch {
        /* ignore */
      }
      const host: Host = { term, fit, shell: session.shell, input: "", ready: false };
      hostsRef.current.set(session.id, host);
      term.onData((d) => handleInput(session.id, d));
      await window.yod.terminal.create({ id: session.id, cwd: session.cwd, shell: session.shell });
      host.ready = true;
    },
    [handleInput]
  );

  const disposeHost = useCallback((id: string) => {
    const host = hostsRef.current.get(id);
    if (!host) return;
    void window.yod.terminal.kill(id);
    try {
      host.term.dispose();
    } catch {
      /* ignore */
    }
    hostsRef.current.delete(id);
  }, []);

  useEffect(() => {
    const offData = window.yod.terminal.onData(({ id, data }) => {
      hostsRef.current.get(id)?.term.write(data);
    });
    const offExit = window.yod.terminal.onExit(({ id }) => {
      const host = hostsRef.current.get(id);
      if (host) {
        host.ready = false;
        host.term.write("\r\n\x1b[90m[התהליך הסתיים]\x1b[0m\r\n");
      }
    });
    return () => {
      offData();
      offExit();
      for (const tid of [...hostsRef.current.keys()]) disposeHost(tid);
    };
  }, [disposeHost]);

  useEffect(() => {
    for (const s of sessions) {
      const el = elsRef.current.get(s.id);
      if (el && !hostsRef.current.has(s.id)) void createHost(s, el);
    }
    for (const id of [...hostsRef.current.keys()]) {
      if (!sessions.some((s) => s.id === id)) disposeHost(id);
    }
  }, [sessions, createHost, disposeHost]);

  useEffect(() => {
    for (const host of hostsRef.current.values()) {
      host.term.options.theme = xtermThemeForName(theme);
    }
  }, [theme]);

  useEffect(() => {
    for (const host of hostsRef.current.values()) {
      host.term.options.fontFamily = fontFamily;
      host.term.options.fontSize = fontSize;
      try {
        host.fit.fit();
      } catch {
        /* ignore */
      }
    }
  }, [fontFamily, fontSize]);

  useEffect(() => {
    if (!visible || !activeId) return;
    const raf = requestAnimationFrame(() => {
      const host = hostsRef.current.get(activeId);
      if (host) {
        try {
          host.fit.fit();
        } catch {
          /* ignore */
        }
        host.term.focus();
      }
    });
    return () => cancelAnimationFrame(raf);
  }, [visible, activeId, sessions]);

  useEffect(() => {
    const el = wrapRef.current;
    if (!el) return;
    const ro = new ResizeObserver(() => {
      const id = activeIdRef.current;
      if (!id) return;
      const host = hostsRef.current.get(id);
      if (host) {
        try {
          host.fit.fit();
        } catch {
          /* ignore */
        }
      }
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const newSession = useCallback((shell: ShellKind): string => {
    const n = ++counterRef.current;
    const id = `t${n}`;
    const title = `${shell === "cmd" ? "CMD" : "PowerShell"} ${n}`;
    setSessions((prev) => [...prev, { id, shell, title, cwd: defaultCwdRef.current }]);
    setActiveId(id);
    return id;
  }, []);

  const closeSession = useCallback((id: string) => {
    setSessions((prev) => {
      const next = prev.filter((s) => s.id !== id);
      setActiveId((cur) => (cur === id ? next[next.length - 1]?.id ?? null : cur));
      return next;
    });
  }, []);

  const waitReady = useCallback((id: string) => {
    return new Promise<void>((resolve) => {
      const t0 = Date.now();
      const tick = () => {
        const host = hostsRef.current.get(id);
        if (host && host.ready) return resolve();
        if (Date.now() - t0 > 3000) return resolve();
        setTimeout(tick, 30);
      };
      tick();
    });
  }, []);

  useImperativeHandle(
    ref,
    () => ({
      newSession,
      async openAndRun({ exe, args, cwd }) {
        let id = activeIdRef.current;
        if (!id || !hostsRef.current.has(id)) id = newSession("powershell");
        await waitReady(id);
        const host = hostsRef.current.get(id);
        if (!host) return;
        const line = (cwd ? cdPrefix(host.shell, cwd) : "") + buildInvoke(host.shell, exe, args);
        host.input = "";
        host.term.write(line + "\r\n");
        await window.yod.terminal.write(id, line + "\r\n");
        host.term.focus();
      },
      interruptActive() {
        const id = activeIdRef.current;
        if (!id) return;
        void window.yod.terminal.write(id, "\x03");
        const host = hostsRef.current.get(id);
        if (host) {
          host.term.write("^C\r\n");
          host.input = "";
        }
      },
      focus() {
        const id = activeIdRef.current;
        if (id) hostsRef.current.get(id)?.term.focus();
      },
    }),
    [newSession, waitReady]
  );

  return (
    <div className="terminal-panel" ref={wrapRef}>
      <div className="term-tabs">
        {sessions.map((s) => (
          <div
            key={s.id}
            className={`term-tab${s.id === activeId ? " active" : ""}`}
            onMouseDown={() => setActiveId(s.id)}
          >
            <span className="term-tab-label">{s.title}</span>
            <button
              type="button"
              className="term-tab-close"
              aria-label="סגור טרמינל"
              title="סגור טרמינל"
              onMouseDown={(e) => {
                e.stopPropagation();
                closeSession(s.id);
              }}
            >
              <Icon name="close" size={12} />
            </button>
          </div>
        ))}
        <div className="term-new">
          <button
            type="button"
            className="term-new-btn"
            title="טרמינל PowerShell חדש"
            onMouseDown={() => newSession("powershell")}
          >
            <Icon name="add" size={14} />
          </button>
          <button
            type="button"
            className="term-new-caret"
            aria-label="בחר מעטפת"
            title="בחר מעטפת"
            onMouseDown={(e) => {
              e.stopPropagation();
              if (shellMenu) {
                setShellMenu(null);
                return;
              }
              const r = e.currentTarget.getBoundingClientRect();
              setShellMenu({ x: r.right, y: r.bottom });
            }}
          >
            <Icon name="expand_more" size={14} />
          </button>
          {shellMenu ? (
            <>
              <div className="term-shell-menu-backdrop" onMouseDown={() => setShellMenu(null)} />
              <div
                className="term-shell-menu"
                style={{ position: "fixed", top: shellMenu.y + 2, right: window.innerWidth - shellMenu.x }}
              >
                <button
                  type="button"
                  onMouseDown={() => {
                    setShellMenu(null);
                    newSession("powershell");
                  }}
                >
                  PowerShell
                </button>
                <button
                  type="button"
                  onMouseDown={() => {
                    setShellMenu(null);
                    newSession("cmd");
                  }}
                >
                  CMD
                </button>
              </div>
            </>
          ) : null}
        </div>
      </div>

      <div className="term-hosts">
        {sessions.length === 0 ? (
          <div className="term-empty">
            אין טרמינל פתוח. לחץ + לפתיחת PowerShell, או הרץ קובץ (F5).
          </div>
        ) : null}
        {sessions.map((s) => (
          <div
            key={s.id}
            className="term-host"
            style={{ display: s.id === activeId ? "block" : "none" }}
            ref={(node) => {
              if (node) elsRef.current.set(s.id, node);
              else elsRef.current.delete(s.id);
            }}
          />
        ))}
      </div>
    </div>
  );
});
