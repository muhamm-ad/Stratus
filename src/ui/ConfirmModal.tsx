import { useAppStore } from "@/store/useAppStore";
import { Dialog } from "@/ui/dialog";

export function ConfirmModal() {
  const modal = useAppStore(s => s.modal);
  const closeModal = useAppStore(s => s.closeModal);

  return (
    <Dialog open={!!modal} onClose={closeModal}>
      {modal && (
        <>
          <div className="text-[16px] font-semibold text-foreground mb-2">{modal.title}</div>
          <div className="text-[13px] text-muted-foreground leading-relaxed mb-5">{modal.body}</div>
          <div className="flex justify-end gap-2.5">
            <button
              onClick={closeModal}
              className="bg-transparent border border-border text-foreground rounded-[6px] px-4 py-2 text-[13px] font-semibold cursor-pointer hover:bg-raised transition-colors"
            >
              {modal.cancelLabel}
            </button>
            <button
              onClick={modal.onConfirm}
              className="border-none rounded-[6px] px-4 py-2 text-[13px] font-semibold text-white cursor-pointer transition-opacity hover:opacity-90"
              style={{ background: modal.danger ? "#DC2626" : "var(--primary)" }}
            >
              {modal.confirmLabel}
            </button>
          </div>
        </>
      )}
    </Dialog>
  );
}
