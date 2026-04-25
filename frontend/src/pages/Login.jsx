import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { login } from '../api/client';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await login(email, password);
      const token = res.data.token;
      const payload = JSON.parse(atob(token.split('.')[1]));
      localStorage.setItem('token', token);
      localStorage.setItem('role', payload.role);
      navigate(payload.role === 'reviewer' ? '/reviewer' : '/merchant');
    } catch (err) {
      setError(err.response?.data?.error || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={styles.page}>
      <div style={styles.card}>
        <div style={styles.header}>
          <h1 style={styles.title}>Sign in</h1>
          <p style={styles.subtitle}>KYC Onboarding Portal</p>
        </div>

        <form onSubmit={handleSubmit} id="login-form">
          {error && <div style={styles.error}>{error}</div>}

          <div style={styles.field}>
            <label htmlFor="login-email" style={styles.label}>Email</label>
            <input
              id="login-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              style={styles.input}
              placeholder="you@company.com"
            />
          </div>

          <div style={styles.field}>
            <label htmlFor="login-password" style={styles.label}>Password</label>
            <input
              id="login-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              style={styles.input}
              placeholder="••••••••"
            />
          </div>

          <button id="login-submit" type="submit" disabled={loading} style={styles.button}>
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
        </form>

        <p style={styles.footer}>
          No account?{' '}
          <Link to="/signup" style={styles.link}>Sign up</Link>
        </p>

        <div style={styles.demo}>
          <p style={styles.demoTitle}>Demo accounts</p>
          <p style={styles.demoLine}>reviewer@kyc.dev</p>
          <p style={styles.demoLine}>merchant.draft@kyc.dev</p>
          <p style={styles.demoLine}>merchant.submitted@kyc.dev</p>
          <p style={styles.demoLine}>merchant.atrisk@kyc.dev</p>
          <p style={styles.demoLine}>merchant.moreinfo@kyc.dev</p>
          <p style={styles.demoLine}>merchant.approved@kyc.dev</p>
          <p style={{...styles.demoLine, marginTop: 4, color: 'var(--text-tertiary)'}}>All passwords are: password123</p>
        </div>
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
  header: {
    marginBottom: 24,
  },
  title: {
    fontSize: 20,
    fontWeight: 600,
    color: 'var(--text-primary)',
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 13,
    color: 'var(--text-secondary)',
    fontFamily: 'var(--font-mono)',
  },
  field: {
    marginBottom: 16,
  },
  label: {
    display: 'block',
    fontSize: 13,
    color: 'var(--text-secondary)',
    marginBottom: 6,
  },
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
  footer: {
    textAlign: 'center',
    fontSize: 13,
    color: 'var(--text-secondary)',
    marginTop: 20,
  },
  link: {
    color: 'var(--accent)',
    textDecoration: 'none',
    fontWeight: 500,
  },
  demo: {
    marginTop: 20,
    paddingTop: 16,
    borderTop: '1px solid var(--border)',
  },
  demoTitle: {
    fontSize: 11,
    color: 'var(--text-tertiary)',
    fontFamily: 'var(--font-mono)',
    marginBottom: 6,
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
  },
  demoLine: {
    fontSize: 12,
    color: 'var(--text-secondary)',
    fontFamily: 'var(--font-mono)',
    lineHeight: 1.7,
  },
};
