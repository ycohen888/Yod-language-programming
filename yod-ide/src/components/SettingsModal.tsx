import type { ReactNode } from "react";
import type { Settings, ThemeName } from "../types";
import { FONT_FAMILIES } from "../lib/settings";
import { Icon } from "./Icon";

type Props = {
  settings: Settings;
  onChange: (next: Settings) => void;
  onReset: () => void;
  onClose: () => void;
};

const THEME_OPTIONS: { value: ThemeName; label: string; swatch: string }[] = [
  { value: "dark", label: "כהה", swatch: "#1e1e1e" },
  { value: "light", label: "בהיר", swatch: "#ffffff" },
  { value: "high-contrast", label: "ניגודיות גבוהה", swatch: "#000000" },
];

/** שורת הגדרה: כותרת + תיאור בצד אחד, בקרה בצד השני. */
function Field({
  label,
  desc,
  htmlFor,
  children,
}: {
  label: string;
  desc?: string;
  htmlFor?: string;
  children: ReactNode;
}) {
  return (
    <div className="settings-field">
      <label className="settings-field-info" htmlFor={htmlFor}>
        <span className="settings-field-label">{label}</span>
        {desc ? <span className="settings-field-desc">{desc}</span> : null}
      </label>
      <div className="settings-field-control">{children}</div>
    </div>
  );
}

/** מתג הפעלה/כיבוי (toggle switch). */
function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      className={`settings-switch${checked ? " on" : ""}`}
      onClick={() => onChange(!checked)}
    >
      <span className="settings-switch-thumb" />
    </button>
  );
}

/** סטפר מספרי (− ערך +). */
function Stepper({
  value,
  min,
  max,
  suffix,
  onChange,
}: {
  value: number;
  min: number;
  max: number;
  suffix?: string;
  onChange: (v: number) => void;
}) {
  const clamp = (n: number) => Math.min(max, Math.max(min, n));
  return (
    <div className="settings-stepper">
      <button
        type="button"
        aria-label="הקטן"
        disabled={value <= min}
        onClick={() => onChange(clamp(value - 1))}
      >
        <Icon name="remove" size={16} />
      </button>
      <span className="settings-stepper-val">
        {value}
        {suffix ? <span className="settings-stepper-suffix">{suffix}</span> : null}
      </span>
      <button
        type="button"
        aria-label="הגדל"
        disabled={value >= max}
        onClick={() => onChange(clamp(value + 1))}
      >
        <Icon name="add" size={16} />
      </button>
    </div>
  );
}

/** מסך הגדרות/העדפות — גופן, נושא, טאב, גלישה, שמירה אוטומטית, סידור-בשמירה. */
export function SettingsModal({ settings, onChange, onReset, onClose }: Props) {
  const set = <K extends keyof Settings>(key: K, value: Settings[K]) =>
    onChange({ ...settings, [key]: value });

  const isCustomFont = FONT_FAMILIES.every((f) => f.value !== settings.fontFamily);

  return (
    <div
      className="palette-backdrop"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="settings-box"
        role="dialog"
        aria-label="הגדרות"
        dir="rtl"
        tabIndex={-1}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
        }}
        ref={(n) => n?.focus()}
      >
        <header className="settings-header">
          <div className="settings-header-title">
            <Icon name="settings" size={18} />
            <span>הגדרות</span>
          </div>
          <button type="button" className="settings-close" aria-label="סגור" onClick={onClose}>
            <Icon name="close" size={18} />
          </button>
        </header>

        <div className="settings-body">
          <section className="settings-section">
            <h3 className="settings-section-title">מראה</h3>

            <Field label="נושא" desc="ערכת צבעים לעורך ולסביבה">
              <div className="settings-seg" role="group" aria-label="נושא">
                {THEME_OPTIONS.map((o) => (
                  <button
                    key={o.value}
                    type="button"
                    className={`settings-seg-btn${settings.theme === o.value ? " active" : ""}`}
                    aria-pressed={settings.theme === o.value}
                    onClick={() => set("theme", o.value)}
                  >
                    <span className="settings-seg-swatch" style={{ background: o.swatch }} />
                    {o.label}
                  </button>
                ))}
              </div>
            </Field>

            <Field label="משפחת גופן" desc="הגופן בעורך הקוד" htmlFor="set-font">
              <select
                id="set-font"
                className="settings-select"
                value={settings.fontFamily}
                onChange={(e) => set("fontFamily", e.target.value)}
              >
                {FONT_FAMILIES.map((f) => (
                  <option key={f.value} value={f.value}>
                    {f.label}
                  </option>
                ))}
                {isCustomFont ? <option value={settings.fontFamily}>מותאם אישית</option> : null}
              </select>
            </Field>

            <Field label="גודל גופן" desc="בפיקסלים (10–28)">
              <Stepper
                value={settings.fontSize}
                min={10}
                max={28}
                suffix="px"
                onChange={(v) => set("fontSize", v)}
              />
            </Field>
          </section>

          <section className="settings-section">
            <h3 className="settings-section-title">עריכה</h3>

            <Field label="גודל טאב" desc="מספר הרווחים בהזחה">
              <Stepper
                value={settings.tabSize}
                min={1}
                max={8}
                onChange={(v) => set("tabSize", v)}
              />
            </Field>

            <Field label="גלישת שורות" desc="שבירת שורות ארוכות בגבול העורך">
              <Toggle checked={settings.wordWrap} onChange={(v) => set("wordWrap", v)} />
            </Field>
          </section>

          <section className="settings-section">
            <h3 className="settings-section-title">שמירה</h3>

            <Field label="שמירה אוטומטית" desc="שמירת קבצים שנייה לאחר הקלדה">
              <Toggle checked={settings.autosave} onChange={(v) => set("autosave", v)} />
            </Field>

            <Field label="סידור קוד בשמירה" desc="מסדר קבצי יוד בכל שמירה">
              <Toggle checked={settings.formatOnSave} onChange={(v) => set("formatOnSave", v)} />
            </Field>
          </section>
        </div>

        <footer className="settings-footer">
          <button type="button" className="settings-reset" onClick={onReset}>
            אפס לברירת מחדל
          </button>
          <button type="button" className="settings-done" onClick={onClose}>
            סיום
          </button>
        </footer>
      </div>
    </div>
  );
}
