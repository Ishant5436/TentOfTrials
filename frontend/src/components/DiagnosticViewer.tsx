import React, { useState } from 'react';
import { DiagnosticMetadata } from './DiagnosticTypes';

import { parseDiagnosticMetadata } from './DiagnosticParser';

export const DiagnosticViewer: React.FC = () => {
  const [jsonInput, setJsonInput] = useState<string>('');
  const [metadata, setMetadata] = useState<DiagnosticMetadata | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleParse = () => {
    try {
      const parsed = parseDiagnosticMetadata(jsonInput);
      setMetadata(parsed);
      setError(null);
    } catch (err: any) {
      setMetadata(null);
      setError(err.message);
    }
  };

  return (
    <div className="diagnostic-viewer" style={{ padding: '20px', fontFamily: 'sans-serif' }}>
      <h2>Build Diagnostic Metadata Viewer</h2>
      <p>Paste your diagnostic JSON below to inspect module build status:</p>
      
      <textarea
        style={{ width: '100%', height: '150px', fontFamily: 'monospace' }}
        value={jsonInput}
        onChange={(e) => setJsonInput(e.target.value)}
        placeholder='{"generated_at": "...", "modules": []}'
      />
      <button onClick={handleParse} style={{ marginTop: '10px', padding: '8px 16px', cursor: 'pointer' }}>
        Parse JSON
      </button>

      {error && (
        <div style={{ marginTop: '20px', padding: '10px', backgroundColor: '#fee', color: '#c00', border: '1px solid #fcc' }}>
          <strong>Error:</strong> {error}
        </div>
      )}

      {metadata && (
        <div style={{ marginTop: '20px' }}>
          <h3>Diagnostic Summary</h3>
          <p>
            <strong>Generated At:</strong> {metadata.generated_at} <br/>
            <strong>Commit:</strong> {metadata.commit} <br/>
            <strong>Logd Artifact:</strong> {metadata.diagnostic_logd ? (
              <span style={{ color: 'green' }}>Found ({metadata.diagnostic_logd})</span>
            ) : (
              <span style={{ color: 'red' }}>Missing</span>
            )}
            <br/>
            <strong>Passed:</strong> {metadata.passed} | <strong>Failed:</strong> {metadata.failed}
          </p>

          <table style={{ width: '100%', borderCollapse: 'collapse', marginTop: '10px' }}>
            <thead>
              <tr style={{ backgroundColor: '#eee', textAlign: 'left' }}>
                <th style={{ padding: '8px', border: '1px solid #ccc' }}>Module</th>
                <th style={{ padding: '8px', border: '1px solid #ccc' }}>Status</th>
                <th style={{ padding: '8px', border: '1px solid #ccc' }}>Duration (s)</th>
                <th style={{ padding: '8px', border: '1px solid #ccc' }}>Artifact Path</th>
              </tr>
            </thead>
            <tbody>
              {metadata.modules.map((mod, idx) => {
                const isFail = mod.status === 'FAIL';
                return (
                  <tr key={idx} style={{ backgroundColor: isFail ? '#fee' : '#fff' }}>
                    <td style={{ padding: '8px', border: '1px solid #ccc' }}>{mod.name}</td>
                    <td style={{ padding: '8px', border: '1px solid #ccc', fontWeight: 'bold', color: isFail ? '#c00' : 'inherit' }}>
                      {mod.status}
                    </td>
                    <td style={{ padding: '8px', border: '1px solid #ccc' }}>{mod.elapsed_seconds}</td>
                    <td style={{ padding: '8px', border: '1px solid #ccc' }}>{mod.artifact || 'None'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};
