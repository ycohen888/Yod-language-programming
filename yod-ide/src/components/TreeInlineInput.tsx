import { useEffect, useRef } from "react";
import { Icon } from "./Icon";

type Props = {
  depth: number;
  isDir: boolean;
  defaultValue: string;
  onCommit: (name: string) => void;
  onCancel: () => void;
};

export function TreeInlineInput({ depth, isDir, defaultValue, onCommit, onCancel }: Props) {
  const ref = useRef<HTMLInputElement>(null);
  const doneRef = useRef(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.focus();
    const extStart = defaultValue.lastIndexOf(".");
    if (!isDir && extStart > 0) {
      el.setSelectionRange(0, extStart);
    } else {
      el.select();
    }
  }, [defaultValue, isDir]);

  const finish = (name: string | null) => {
    if (doneRef.current) return;
    doneRef.current = true;
    if (name) onCommit(name);
    else onCancel();
  };

  return (
    <div className="tree-item tree-item-edit" style={{ paddingInlineStart: 8 + depth * 14 }}>
      <span className="chev">
        {isDir ? (
          <Icon name="folder" size={16} className="folder-kind" />
        ) : (
          <Icon name="insert_drive_file" size={16} className="file-kind" />
        )}
      </span>
      <input
        ref={ref}
        className="tree-inline-input"
        defaultValue={defaultValue}
        spellCheck={false}
        onBlur={() => {
          const v = ref.current?.value.trim() ?? "";
          finish(v || null);
        }}
        onKeyDown={(e) => {
          e.stopPropagation();
          if (e.key === "Enter") {
            e.preventDefault();
            const v = (e.target as HTMLInputElement).value.trim();
            finish(v || null);
          } else if (e.key === "Escape") {
            e.preventDefault();
            doneRef.current = true;
            onCancel();
          }
        }}
        onClick={(e) => e.stopPropagation()}
      />
    </div>
  );
}
