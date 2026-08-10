// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		interface Locals {
			// Populated by hooks.server.ts for authenticated requests so that
			// role guards (hooks) and layout loads share one /api/me call.
			user?: {
				id: number;
				email: string;
				name: string;
				role: string;
				roles: string[];
			} | null;
		}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
