import type { DiagnosticMetadata } from './DiagnosticTypes.ts';

export function parseDiagnosticMetadata(jsonString: string): DiagnosticMetadata {
  let parsed: any;
  try {
    parsed = JSON.parse(jsonString);
  } catch (err: any) {
    throw new Error("Invalid JSON: " + err.message);
  }

  if (!parsed || typeof parsed !== 'object') {
    throw new Error("Invalid format: root must be a JSON object.");
  }

  if (!parsed.modules || !Array.isArray(parsed.modules)) {
    throw new Error("Invalid format: 'modules' array is missing.");
  }

  return parsed as DiagnosticMetadata;
}
