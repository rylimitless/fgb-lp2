type ToastVariant = "default" | "success" | "warning" | "error";

export type ToastMessage = {
	id: number;
	title?: string;
	message: string;
	variant?: ToastVariant;
};

export const toasts = $state<ToastMessage[]>([]);

export function showToast(message: string, options: { title?: string; variant?: ToastVariant } = {}) {
	const id = Date.now() + Math.floor(Math.random() * 1000);
	toasts.push({ id, message, title: options.title, variant: options.variant ?? "default" });
	setTimeout(() => dismissToast(id), 4200);
	return id;
}

export function dismissToast(id: number) {
	const index = toasts.findIndex((toast) => toast.id === id);
	if (index !== -1) toasts.splice(index, 1);
}
