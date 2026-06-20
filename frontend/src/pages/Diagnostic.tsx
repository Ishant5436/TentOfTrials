import React from 'react';
import { DiagnosticViewer } from '../components/DiagnosticViewer';

const Diagnostic: React.FC = () => {
  return (
    <div className="page-container">
      <header className="page-header">
        <h1>Diagnostic Metadata Viewer</h1>
        <p>Inspect build diagnostic JSON metadata.</p>
      </header>
      <div className="page-content">
        <DiagnosticViewer />
      </div>
    </div>
  );
};

export default Diagnostic;
