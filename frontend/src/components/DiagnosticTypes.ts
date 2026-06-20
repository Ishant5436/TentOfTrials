export interface DiagnosticModule {
  name: string;
  status: 'PASS' | 'FAIL' | 'SKIP' | string;
  elapsed_seconds: number;
  artifact: string | null;
  output: string;
}

export interface DiagnosticMetadata {
  generated_at: string;
  commit: string;
  diagnostic_logd: string | null;
  diagnostic_logd_error: string | null;
  chunked: boolean;
  chunk_size_bytes: number | null;
  password?: string;
  decrypt_command?: string;
  total_modules: number;
  passed: number;
  failed: number;
  modules: DiagnosticModule[];
  pr_note?: string;
}
