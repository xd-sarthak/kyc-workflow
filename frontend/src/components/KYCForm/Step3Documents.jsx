import { useRef } from 'react';

const FILE_TYPES = [
  { key: 'pan', label: 'PAN Card' },
  { key: 'aadhaar', label: 'Aadhaar Card' },
  { key: 'bank_statement', label: 'Bank Statement' },
];

const ACCEPTED = '.pdf,.jpg,.jpeg,.png';
const MAX_SIZE = 5 * 1024 * 1024;

export default function Step3Documents({ files, onChange, onBack, onSubmit, loading }) {
  const fileRefs = useRef({});
  const [errors, setErrors] = useState({});

  const handleFile = (key, file) => {
    if (file) {
      const valid = ['application/pdf', 'image/jpeg', 'image/png'];
      if (!valid.includes(file.type)) {
        setErrors({ ...errors, [key]: 'Invalid file type. Only PDF/JPG/PNG allowed.' });
        return;
      }
      if (file.size > MAX_SIZE) {
        setErrors({ ...errors, [key]: 'File too large. Max 5 MB.' });
        return;
      }
    }
    setErrors({ ...errors, [key]: undefined });
    onChange({ ...files, [key]: file });
  };

  const removeFile = (key) => {
    onChange({ ...files, [key]: null });
    setErrors({ ...errors, [key]: undefined });
    if (fileRefs.current[key]) fileRefs.current[key].value = '';
  };

  const allUploaded = FILE_TYPES.every((ft) => files[ft.key]);

  return (
    <div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {FILE_TYPES.map((ft) => (
          <div key={ft.key}>
            <label style={labelStyle}>{ft.label}</label>
            {files[ft.key] ? (
              <div style={fileInfoStyle}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <p style={{ fontSize: 13, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {files[ft.key].name}
                  </p>
                  <p style={{ fontSize: 12, color: 'var(--text-tertiary)', fontFamily: 'var(--font-mono)', marginTop: 2 }}>
                    {(files[ft.key].size / 1024).toFixed(1)} KB
                  </p>
                </div>
                <button
                  onClick={() => removeFile(ft.key)}
                  style={removeBtnStyle}
                  aria-label={`Remove ${ft.label}`}
                >
                  ✕
                </button>
              </div>
            ) : (
              <div>
                <div
                  style={{ ...dropzoneStyle, borderColor: errors[ft.key] ? 'var(--danger)' : 'var(--border)' }}
                  onClick={() => fileRefs.current[ft.key]?.click()}
                >
                  <p style={{ fontSize: 13, color: 'var(--text-tertiary)' }}>
                    Click to upload
                  </p>
                  <p style={{ fontSize: 11, color: 'var(--text-tertiary)', fontFamily: 'var(--font-mono)', marginTop: 4 }}>
                    PDF, JPEG, PNG · Max 5 MB
                  </p>
                  <input
                    ref={(el) => (fileRefs.current[ft.key] = el)}
                    id={`doc-${ft.key}`}
                    type="file"
                    accept={ACCEPTED}
                    onChange={(e) => handleFile(ft.key, e.target.files[0])}
                    style={{ display: 'none' }}
                  />
                </div>
                {errors[ft.key] && (
                  <p style={{ fontSize: 12, color: 'var(--danger)', marginTop: 4, fontFamily: 'var(--font-mono)' }}>
                    {errors[ft.key]}
                  </p>
                )}
              </div>
            )}
          </div>
        ))}
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 24 }}>
        <button id="step3-back" onClick={onBack} style={backBtnStyle}>Back</button>
        <button
          id="step3-submit"
          onClick={onSubmit}
          disabled={!allUploaded || loading}
          style={{
            ...submitBtnStyle,
            opacity: (!allUploaded || loading) ? 0.4 : 1,
            cursor: (!allUploaded || loading) ? 'not-allowed' : 'pointer',
          }}
        >
          {loading ? 'Submitting…' : 'Submit for review'}
        </button>
      </div>
    </div>
  );
}

const labelStyle = { display: 'block', fontSize: 13, color: 'var(--text-secondary)', marginBottom: 6 };

const dropzoneStyle = {
  border: '1px dashed var(--border)',
  borderRadius: 'var(--radius)',
  padding: '20px 16px',
  textAlign: 'center',
  cursor: 'pointer',
  transition: 'border-color 150ms ease',
};

const fileInfoStyle = {
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  padding: '10px 12px',
  background: 'var(--bg-input)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius)',
};

const removeBtnStyle = {
  width: 28,
  height: 28,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  background: 'transparent',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius)',
  color: 'var(--text-secondary)',
  fontSize: 12,
  cursor: 'pointer',
  flexShrink: 0,
  transition: 'border-color 150ms ease',
};

const backBtnStyle = {
  padding: '10px 24px',
  background: 'transparent',
  color: 'var(--text-secondary)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius)',
  fontSize: 14,
  cursor: 'pointer',
};

const submitBtnStyle = {
  padding: '10px 24px',
  background: 'var(--accent)',
  color: '#0A0A0A',
  border: 'none',
  borderRadius: 'var(--radius)',
  fontSize: 14,
  fontWeight: 600,
  transition: 'opacity 150ms ease',
};
