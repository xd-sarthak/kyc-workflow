import { useState } from 'react';

export default function Step1Personal({ data, onChange, onNext }) {
  const [errors, setErrors] = useState({});

  const validate = () => {
    const e = {};
    if (!data.full_name?.trim()) e.full_name = 'Required';
    if (!data.email?.trim()) e.email = 'Required';
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(data.email)) e.email = 'Invalid email';
    if (!data.phone?.trim()) e.phone = 'Required';
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const handleNext = () => { if (validate()) onNext(); };

  return (
    <div>
      <Field
        id="pd-full-name"
        label="Full name"
        type="text"
        value={data.full_name || ''}
        onChange={(v) => onChange({ ...data, full_name: v })}
        error={errors.full_name}
        placeholder="Jane Doe"
      />
      <Field
        id="pd-email"
        label="Email address"
        type="email"
        value={data.email || ''}
        onChange={(v) => onChange({ ...data, email: v })}
        error={errors.email}
        placeholder="jane@company.com"
      />
      <Field
        id="pd-phone"
        label="Phone number"
        type="tel"
        value={data.phone || ''}
        onChange={(v) => onChange({ ...data, phone: v })}
        error={errors.phone}
        placeholder="+91 98765 43210"
      />
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 24 }}>
        <button id="step1-next" onClick={handleNext} style={btnStyle}>
          Continue
        </button>
      </div>
    </div>
  );
}

function Field({ id, label, type, value, onChange, error, placeholder }) {
  return (
    <div style={{ marginBottom: 16 }}>
      <label htmlFor={id} style={labelStyle}>{label}</label>
      <input
        id={id}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        style={{
          ...inputStyle,
          borderColor: error ? 'var(--danger)' : 'var(--border)',
        }}
      />
      {error && <p style={errorStyle}>{error}</p>}
    </div>
  );
}

const labelStyle = {
  display: 'block',
  fontSize: 13,
  color: 'var(--text-secondary)',
  marginBottom: 6,
};

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

const errorStyle = {
  fontSize: 12,
  color: 'var(--danger)',
  marginTop: 4,
  fontFamily: 'var(--font-mono)',
};

const btnStyle = {
  padding: '10px 24px',
  background: 'var(--accent)',
  color: '#0A0A0A',
  border: 'none',
  borderRadius: 'var(--radius)',
  fontSize: 14,
  fontWeight: 600,
  fontFamily: 'var(--font-body)',
  cursor: 'pointer',
  transition: 'opacity 150ms ease',
};
