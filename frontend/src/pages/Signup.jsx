import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { signup } from '../api/client';

export default function Signup() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState('merchant');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await signup(email, password, role);
      const token = res.data.token;
      const payload = JSON.parse(atob(token.split('.')[1]));
      localStorage.setItem('token', token);
      localStorage.setItem('role', payload.role);
      navigate(payload.role === 'reviewer' ? '/reviewer' : '/merchant');
    } catch (err) {
      setError(err.response?.data?.error || 'Signup failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={styles.page}>
      <div style={styles.card}>
        <div style={styles.header}>
          <h1 style={styles.title}>Create account</h1>
          <p style={styles.subtitle}>KYC Onboarding Portal</p>
        </div>

        <form onSubmit={handleSubmit} id="signup-form">
          {error && <div style={styles.error}>{error}</div>}

          <div style={styles.field}>
            <label htmlFor="signup-email" style={styles.label}>Email</label>
            <input
              id="signup-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              style={styles.input}
              placeholder="you@company.com"
            />
          </div>

          <div style={styles.field}>
            <label htmlFor="signup-password" style={styles.label}>Password</label>
            <input
              id="signup-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={6}
              style={styles.input}
              placeholder="Min 6 characters"
            />
          </div>

          <div style={styles.field}>
            <label style={styles.label}>Role</label>
            <div style={styles.roleGroup}>
              {['merchant', 'reviewer'].map((r) => (
                <button
                  key={r}
                  type="button"
                  id={`signup-role-${r}`}
                  onClick={() => setRole(r)}
                  style={{
                    ...styles.roleBtn,
                    background: role === r ? 'var(--accent-dim)' : 'var(--bg-input)',
                    borderColor: role === r ? 'var(--accent)' : 'var(--border)',
                    color: role === r ? 'var(--accent)' : 'var(--text-secondary)',
                  }}
                >
                  {r}
                </button>
              ))}
            </div>
          </div>

          <button id="signup-submit" type="submit" disabled={loading} style={styles.button}>
            {loading ? 'Creating…' : 'Create account'}
          </button>
        </form>

        <p style={styles.footer}>
          Have an account?{' '}
          <Link to="/login" style={styles.link}>Sign in</Link>
        </p>
      </div>
    </div>
  );
}

const styles = {
  page: {
    minHeight: '100vh',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 16,
    background: 'var(--bg-base)',
  },
  card: {
    width: '100%',
    maxWidth: 400,
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 'var(--radius)',
    padding: 24,
  },
  header: { marginBottom: 24 },
  title: { fontSize: 20, fontWeight: 600, color: 'var(--text-primary)', marginBottom: 4 },
  subtitle: { fontSize: 13, color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)' },
  field: { marginBottom: 16 },
  label: { display: 'block', fontSize: 13, color: 'var(--text-secondary)', marginBottom: 6 },
  input: {
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
  },
  roleGroup: { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 },
  roleBtn: {
    padding: '10px 12px',
    border: '1px solid var(--border)',
    borderRadius: 'var(--radius)',
    fontSize: 13,
    fontFamily: 'var(--font-mono)',
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'all 150ms ease',
    textTransform: 'capitalize',
  },
  button: {
    width: '100%',
    padding: '10px 16px',
    background: 'var(--accent)',
    color: '#0A0A0A',
    border: 'none',
    borderRadius: 'var(--radius)',
    fontSize: 14,
    fontWeight: 600,
    fontFamily: 'var(--font-body)',
    cursor: 'pointer',
    transition: 'opacity 150ms ease',
    marginTop: 8,
  },
  error: {
    padding: '10px 12px',
    background: 'var(--danger-dim)',
    borderLeft: '2px solid var(--danger)',
    borderRadius: 'var(--radius)',
    fontSize: 13,
    color: 'var(--danger)',
    marginBottom: 16,
  },
  footer: { textAlign: 'center', fontSize: 13, color: 'var(--text-secondary)', marginTop: 20 },
  link: { color: 'var(--accent)', textDecoration: 'none', fontWeight: 500 },
};
