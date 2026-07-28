/**
 * FGB Academy — toast store.
 *
 * Toasts are announced via `role="status" aria-live="polite"` on the viewport.
 * Every toast can optionally carry a secondary action button (e.g., "Undo" on
 * destructive mutations). The action fires and the toast dismisses; callers
 * that want a non-dismissing action should manage state elsewhere.
 */
type ToastVariant = "default" | "success" | "warning" | "error";

export type ToastAction = {
	label: string;
	onClick: () => void;
};

export type ToastMessage = {
	id: number;
	title?: string;
	message: string;
	variant?: ToastVariant;
	action?: ToastAction;
	/** Milliseconds until auto-dismiss. Set to 0 to keep the toast until manually dismissed. */
	duration?: number;
};

export type ToastOptions = {
	title?: string;
	variant?: ToastVariant;
	action?: ToastAction;
	duration?: number;
};

export const toasts = $state<ToastMessage[]>([]);

const DEFAULT_DURATION = 4200;
const ACTION_DURATION = 7000;

export function showToast(message: string, options: ToastOptions = {}) {
	const id = Date.now() + Math.floor(Math.random() * 1000);
	const duration =
		options.duration ??
		(options.action ? ACTION_DURATION : DEFAULT_DURATION);
	toasts.push({
		id,
		message,
		title: options.title,
		variant: options.variant ?? "default",
		action: options.action,
		duration,
	});
	if (duration > 0) {
		setTimeout(() => dismissToast(id), duration);
	}
	return id;
}

export function dismissToast(id: number) {
	const index = toasts.findIndex((toast) => toast.id === id);
	if (index !== -1) toasts.splice(index, 1);
}
