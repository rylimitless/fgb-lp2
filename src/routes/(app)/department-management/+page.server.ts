import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export type Department = {
    id: number;
    name: string;
    created_at: string;
};

export type DepartmentUser = {
    id: number;
    email: string;
    name: string;
    role: string;
    created_at: string;
};

export const load: PageServerLoad = async (event) => {
    const res = await apiFetch(event, "/api/admin/departments");
    const departments: Department[] = res.ok ? await res.json() : [];
    return { departments };
};
