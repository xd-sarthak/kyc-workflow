import { useState } from 'react';

const BUSINESS_TYPES = ['retail', 'wholesale', 'services', 'manufacturing', 'technology', 'other'];

export default function Step2Business({ data, onChange, onNext, onBack }) {
  const [errors, setErrors] = useState({});

  const validate = () => {
    const e = {};
    if (!data.business_name?.trim()) e.business_name = 'Required';
    if (!data.business_type?.trim()) e.business_type = 'Required';
    if (!data.expected_monthly_volume || data.expected_monthly_volume <= 0)
      e.expected_monthly_volume = 'Must be greater than 0';
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleNext = () => { if (validate()) onNext(); };

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <label htmlFor="bd-name" style={labelStyle}>Business name</label>
        <input
          id="bd-name"
          type="text"
          value={data.business_name || ''}
          onChange={(e) => onChange({ ...data, business_name: e.target.value })}
          placeholder="Acme Inc."
          style={{ ...inputStyle, borderColor: errors.business_name ? 'var(--danger)' : 'var(--border)' }}
        />
        {errors.business_name && <p style={errorStyle}>{errors.business_name}</p>}
      </div>

      <div style={{ marginBottom: 16 }}>
        <label htmlFor="bd-type" style={labelStyle}>Business type</label>
        <select
          id="bd-type"
          value={data.business_type || ''}
          onChange={(e) => onChange({ ...data, business_type: e.target.value })}
          style={{ ...inputStyle, appearance: 'none', borderColor: errors.business_type ? 'var(--danger)' : 'var(--border)' }}
        >
          <option value="" style={{ background: '#111' }}>Select type</option>
          {BUSINESS_TYPES.map((t) => (
            <option key={t} value={t} style={{ background: '#111', textTransform: 'capitalize' }}>
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </option>
          ))}
        </select>
        {errors.business_type && <p style={errorStyle}>{errors.business_type}</p>}
      </div>

      <div style={{ marginBottom: 16 }}>
        <label htmlFor="bd-volume" style={labelStyle}>Expected monthly volume (₹)</label>
        <input
          id="bd-volume"
          type="number"
          min="1"
          value={data.expected_monthly_volume || ''}
          onChange={(e) => onChange({ ...data, expected_monthly_volume: parseFloat(e.target.value) || 0 })}
          placeholder="50000"
          style={{ ...inputStyle, borderColor: errors.expected_monthly_volume ? 'var(--danger)' : 'var(--border)' }}
        />
        {errors.expected_monthly_volume && <p style={errorStyle}>{errors.expected_monthly_volume}</p>}
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 24 }}>
        <button id="step2-back" onClick={onBack} style={backBtnStyle}>Back</button>
        <button id="step2-next" onClick={handleNext} style={btnStyle}>Continue</button>
      </div>
    </div>
  );
}

const labelStyle = { display: 'block', fontSize: 13, color: 'var(--text-secondary)', marginBottom: 6 };

const inputStyle = {
  width: '100%',
  padding: '10px 12px',
  background: 'var(--bg-input)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius)',
  color: 'var(--text-primary)',
  fontSize: 14,
  fontFamily: 'var(--font-body)',
  outline: 'none',
  transition: 'border-color 150ms ease',
};

const errorStyle = { fontSize: 12, color: 'var(--danger)', marginTop: 4, fontFamily: 'var(--font-mono)' };

const btnStyle = {
  padding: '10px 24px',
  background: 'var(--accent)',
  color: '#0A0A0A',
  border: 'none',
  borderRadius: 'var(--radius)',
  fontSize: 14,
  fontWeight: 600,
  cursor: 'pointer',
  transition: 'opacity 150ms ease',
};

const backBtnStyle = {
  padding: '10px 24px',
  background: 'transparent',
  color: 'var(--text-secondary)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius)',
  fontSize: 14,
  cursor: 'pointer',
  transition: 'border-color 150ms ease',
};
