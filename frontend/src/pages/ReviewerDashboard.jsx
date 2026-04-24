import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { getQueue, getSubmissionDetail, transitionSubmission, getMetrics } from '../api/client';
import { useToast } from '../App';

export default function ReviewerDashboard() {
  const [queue, setQueue] = useState([]);
  const [total, setTotal] = useState(0);
  const [metrics, setMetrics] = useState(null);
  const [selected, setSelected] = useState(null);
  const [note, setNote] = useState('');
  const [transitioning, setTransitioning] = useState(false);
  const navigate = useNavigate();
  const toast = useToast();

  const fetchQueue = useCallback(async () => {
    try {
      const res = await getQueue();
      setQueue(res.data.submissions || []);
      setTotal(res.data.total);
    } catch { /* ignore */ }
  }, []);

  const fetchMetrics = useCallback(async () => {
    try {
      const res = await getMetrics();
      setMetrics(res.data);
    } catch { /* ignore */ }
  }, []);

  useEffect(() => { fetchQueue(); fetchMetrics(); }, [fetchQueue, fetchMetrics]);

  const handleSelect = async (id) => {
    try {
      const res = await getSubmissionDetail(id);
      setSelected(res.data);
      setNote('');
    } catch {
      toast('Failed to load submission');
    }
  };

  const handleTransition = async (toState) => {
    if (!selected) return;
    if ((toState === 'rejected' || toState === 'more_info_requested') && !note.trim()) {
      toast('Note is required for this action');
      return;
    }
    setTransitioning(true);
    try {
      await transitionSubmission(selected.submission_id, toState, note);
      toast(`Transitioned to ${toState.replace(/_/g, ' ')}`, 'success');
      setSelected(null);
      await fetchQueue();
      await fetchMetrics();
    } catch (err) {
      toast(err.response?.data?.error || 'Transition failed');
    } finally {
      setTransitioning(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('role');
    navigate('/login');
  };

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-base)' }}>
      {/* Nav */}
      <nav style={navStyle}>
        <div style={navInner}>
          <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>KYC Reviewer</span>
          <button id="reviewer-logout" onClick={logout} style={navBtnStyle}>Sign out</button>
        </div>
      </nav>

      <div style={{ maxWidth: 1100, margin: '0 auto', padding: '32px 16px' }}>
        {/* Metrics bar */}
        {metrics && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 32 }}>
            <MetricBlock label="Queue size" value={metrics.queue_size} />
            <MetricBlock label="Avg wait" value={fmtDuration(metrics.avg_time_in_queue_seconds)} />
            <MetricBlock label="Approval rate (7d)" value={`${(metrics.approval_rate_last_7d * 100).toFixed(1)}%`} />
          </div>
        )}

        {selected ? (
          /* === DETAIL VIEW === */
          <div>
            <button
              onClick={() => setSelected(null)}
              style={{ ...navBtnStyle, marginBottom: 20, fontSize: 13 }}
            >
              ← Back to queue
            </button>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 16, alignItems: 'start' }}>
              {/* Left: submission data */}
              <div style={cardStyle}>
                <div style={{ marginBottom: 20 }}>
                  <p style={monoLabelStyle}>Submission</p>
                  <p style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-tertiary)', marginTop: 2 }}>
                    {selected.submission_id}
                  </p>
                </div>

                {selected.personal_details && (
                  <DetailSection title="Personal details">
                    <DetailRow label="Name" value={selected.personal_details.full_name} />
                    <DetailRow label="Email" value={selected.personal_details.email} />
                    <DetailRow label="Phone" value={selected.personal_details.phone} />
                  </DetailSection>
                )}

                {selected.business_details && (
                  <DetailSection title="Business details">
                    <DetailRow label="Name" value={selected.business_details.business_name} />
                    <DetailRow label="Type" value={selected.business_details.business_type} />
                    <DetailRow label="Volume" value={`₹${selected.business_details.expected_monthly_volume?.toLocaleString()}`} />
                  </DetailSection>
                )}

                {selected.documents?.length > 0 && (
                  <DetailSection title="Documents" noBorder>
                    {selected.documents.map((doc) => (
                      <div key={doc.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '6px 0' }}>
                        <span style={{ fontSize: 13, color: 'var(--text-primary)', textTransform: 'capitalize' }}>
                          {doc.file_type.replace('_', ' ')}
                        </span>
                        <span style={{ fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)' }}>
                          {doc.original_name}
                        </span>
                      </div>
                    ))}
                  </DetailSection>
                )}
              </div>

              {/* Right: action panel */}
              <div style={{ ...cardStyle, position: 'sticky', top: 16 }}>
                {/* State badge */}
                <div style={{ marginBottom: 20 }}>
                  <p style={monoLabelStyle}>Current state</p>
                  <span style={{
                    display: 'inline-block',
                    marginTop: 6,
                    padding: '4px 10px',
                    border: `1px solid var(--border)`,
                    borderRadius: 'var(--radius)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 12,
                    fontWeight: 500,
                    color: 'var(--text-secondary)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                  }}>
                    {selected.state.replace(/_/g, ' ')}
                  </span>
                </div>

                {/* Previous note */}
                {selected.reviewer_note && (
                  <div style={{ ...noteBoxStyle, marginBottom: 16 }}>
                    <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', marginBottom: 4, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                      Previous note
                    </p>
                    <p style={{ fontSize: 13, color: 'var(--text-primary)' }}>{selected.reviewer_note}</p>
                  </div>
                )}

                {/* Note textarea */}
                <div style={{ marginBottom: 16 }}>
                  <label htmlFor="reviewer-note" style={{ display: 'block', fontSize: 13, color: 'var(--text-secondary)', marginBottom: 6 }}>
                    Reviewer note
                  </label>
                  <textarea
                    id="reviewer-note"
                    value={note}
                    onChange={(e) => setNote(e.target.value)}
                    rows={3}
                    placeholder="Required for reject or request info"
                    style={{
                      width: '100%',
                      padding: '10px 12px',
                      background: 'var(--bg-input)',
                      border: '1px solid var(--border)',
                      borderRadius: 'var(--radius)',
                      color: 'var(--text-primary)',
                      fontSize: 13,
                      fontFamily: 'var(--font-body)',
                      outline: 'none',
                      resize: 'vertical',
                      transition: 'border-color 150ms ease',
                    }}
                  />
                </div>

                {/* Action buttons */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  {selected.state === 'submitted' && (
                    <ActionBtn id="btn-under-review" label="Start review" color="var(--accent)" bg onClick={() => handleTransition('under_review')} disabled={transitioning} />
                  )}
                  {selected.state === 'under_review' && (
                    <>
                      <ActionBtn id="btn-approve" label="Approve" color="var(--success)" onClick={() => handleTransition('approved')} disabled={transitioning} />
                      <ActionBtn id="btn-reject" label="Reject" color="var(--danger)" onClick={() => handleTransition('rejected')} disabled={transitioning} />
                      <ActionBtn id="btn-more-info" label="Request info" color="var(--text-secondary)" onClick={() => handleTransition('more_info_requested')} disabled={transitioning} />
                    </>
                  )}
                </div>
              </div>
            </div>
          </div>
        ) : (
          /* === QUEUE TABLE === */
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <h2 style={{ fontSize: 16, fontWeight: 600, color: 'var(--text-primary)' }}>Review queue</h2>
              <span style={{ fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)' }}>{total} total</span>
            </div>

            {queue.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '64px 0' }}>
                <p style={{ fontFamily: 'var(--font-mono)', fontSize: 13, color: 'var(--text-tertiary)' }}>No submissions in queue</p>
              </div>
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    {['Merchant', 'Submitted', 'Time in queue', 'Status', ''].map((h) => (
                      <th key={h} style={thStyle}>{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {queue.map((item) => (
                    <tr
                      key={item.submission_id}
                      id={`queue-item-${item.submission_id}`}
                      style={{
                        borderLeft: item.at_risk ? '2px solid var(--danger)' : '2px solid transparent',
                      }}
                    >
                      <td style={tdStyle}>
                        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 13, color: 'var(--text-primary)' }}>
                          {item.merchant_id.slice(0, 8)}
                        </span>
                      </td>
                      <td style={tdStyle}>
                        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: 'var(--text-tertiary)' }}>
                          {new Date(item.created_at).toLocaleDateString()}
                        </span>
                      </td>
                      <td style={tdStyle}>
                        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, color: item.at_risk ? 'var(--danger)' : 'var(--text-tertiary)' }}>
                          {fmtDuration((Date.now() - new Date(item.created_at).getTime()) / 1000)}
                        </span>
                      </td>
                      <td style={tdStyle}>
                        <span style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-secondary)',
                          textTransform: 'uppercase',
                          letterSpacing: '0.06em',
                        }}>
                          {item.state.replace(/_/g, ' ')}
                        </span>
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right' }}>
                        <button
                          onClick={() => handleSelect(item.submission_id)}
                          style={reviewBtnStyle}
                        >
                          Review →
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}
      </div>

      {/* Responsive override */}
      <style>{`
        @media (max-width: 768px) {
          [style*="gridTemplateColumns: 1fr 340px"] {
            grid-template-columns: 1fr !important;
          }
          [style*="position: sticky"] {
            position: static !important;
          }
        }
      `}</style>
    </div>
  );
}

function MetricBlock({ label, value }) {
  return (
    <div style={{
      background: 'var(--bg-card)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius)',
      padding: 20,
    }}>
      <p style={{ fontFamily: 'var(--font-mono)', fontSize: 24, fontWeight: 600, color: 'var(--text-primary)' }}>{value}</p>
      <p style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 4 }}>{label}</p>
    </div>
  );
}

function DetailSection({ title, children, noBorder }) {
  return (
    <div style={{ paddingBottom: noBorder ? 0 : 16, marginBottom: noBorder ? 0 : 16, borderBottom: noBorder ? 'none' : '1px solid var(--border)' }}>
      <p style={monoLabelStyle}>{title}</p>
      {children}
    </div>
  );
}

function DetailRow({ label, value }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 0' }}>
      <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ fontSize: 13, color: 'var(--text-primary)' }}>{value || '—'}</span>
    </div>
  );
}

function ActionBtn({ id, label, color, bg, onClick, disabled }) {
  return (
    <button
      id={id}
      onClick={onClick}
      disabled={disabled}
      style={{
        padding: '10px 16px',
        background: bg ? 'var(--accent)' : 'transparent',
        color: bg ? '#0A0A0A' : color,
        border: bg ? 'none' : `1px solid ${color}`,
        borderRadius: 'var(--radius)',
        fontSize: 13,
        fontWeight: 500,
        fontFamily: 'var(--font-body)',
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.5 : 1,
        transition: 'opacity 150ms ease',
        textAlign: 'center',
      }}
    >
      {label}
    </button>
  );
}

function fmtDuration(seconds) {
  if (!seconds || seconds <= 0) return '0m';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (h > 24) return `${Math.floor(h / 24)}d ${h % 24}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

const navStyle = { borderBottom: '1px solid var(--border)', background: 'var(--bg-card)' };
const navInner = { maxWidth: 1100, margin: '0 auto', padding: '12px 16px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' };
const navBtnStyle = { background: 'transparent', border: 'none', color: 'var(--text-secondary)', fontSize: 13, cursor: 'pointer', fontFamily: 'var(--font-body)' };
const cardStyle = { background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: 24 };
const monoLabelStyle = { fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 8 };
const noteBoxStyle = { background: 'var(--bg-input)', border: '1px solid var(--border)', borderLeft: '2px solid var(--danger)', borderRadius: 'var(--radius)', padding: '12px 16px' };
const thStyle = { textAlign: 'left', fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.06em', padding: '8px 12px', borderBottom: '1px solid var(--border)' };
const tdStyle = { padding: '12px', borderBottom: '1px solid var(--border)', verticalAlign: 'middle' };
const reviewBtnStyle = { background: 'transparent', border: 'none', color: 'var(--accent)', fontSize: 13, fontWeight: 500, cursor: 'pointer', fontFamily: 'var(--font-body)', transition: 'opacity 150ms ease' };
