import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { getQueue, getSubmissionDetail, transitionSubmission, getMetrics } from '../api/client';
import { useToast } from '../App';

export default function ReviewerDashboard() {
  const [queue, setQueue] = useState([]);
  const [metrics, setMetrics] = useState({ queue_size: 0, average_wait_hours: 0, approval_rate_7d: 0 });
  const [selected, setSelected] = useState(null);
  const [note, setNote] = useState('');
  const [noteError, setNoteError] = useState('');
  const [loading, setLoading] = useState(true);
  const [transitioning, setTransitioning] = useState(false);
  const [docView, setDocView] = useState(null); // { url, type }
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(0);
  const limit = 20;

  const navigate = useNavigate();
  const toast = useToast();

  const fetchQueue = useCallback(async () => {
    try {
      const res = await getQueue(limit, page * limit);
      setQueue(res.data.submissions || []);
      setTotal(res.data.total || 0);
    } catch {
      toast('Failed to load queue');
    }
  }, [page, limit, toast]);

  const fetchMetrics = useCallback(async () => {
    try {
      const res = await getMetrics();
      setMetrics({
        queue_size: res.data.queue_size || 0,
        average_wait_hours: (res.data.avg_time_in_queue_seconds || 0) / 3600,
        approval_rate_7d: res.data.approval_rate_last_7d || 0
      });
    } catch {
      // Ignore
    }
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchQueue(), fetchMetrics()]);
    setLoading(false);
  }, [fetchQueue, fetchMetrics]);

  useEffect(() => { loadData(); }, [loadData]);

  const handleReviewClick = async (id) => {
    try {
      const res = await getSubmissionDetail(id);
      setSelected(res.data);
      setNote('');
      setNoteError('');
    } catch (err) {
      toast(err.response?.data?.error || 'Failed to load details');
    }
  };

  const handleTransition = async (toState) => {
    if (!selected) return;
    
    // Note validation
    setNoteError('');
    if ((toState === 'rejected' || toState === 'more_info_requested') && !note.trim()) {
      setNoteError('A reason is required.');
      return;
    }
    
    setTransitioning(true);
    try {
      await transitionSubmission(selected.submission_id, toState, note);
      toast(`Transitioned to ${toState.replace(/_/g, ' ')}`, 'success');

      if (toState === 'under_review') {
        // Stay on detail view, re-fetch to show updated state and action buttons
        const res = await getSubmissionDetail(selected.submission_id);
        setSelected(res.data);
        setNote('');
      } else {
        // Terminal action (approve/reject/more_info) — go back to queue
        setSelected(null);
      }
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

  if (loading && !queue.length) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-base)' }}>
        <p style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', fontSize: 13 }}>Loading…</p>
      </div>
    );
  }

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-base)' }}>
      {/* Document Viewer Modal/Lightbox */}
      {docView && (
        <div style={{ position: 'fixed', inset: 0, zIndex: 9999, background: 'rgba(0,0,0,0.8)', display: 'flex', alignItems: 'center', justifyContent: 'center' }} onClick={() => setDocView(null)}>
          <button style={{ position: 'absolute', top: 20, right: 20, background: 'transparent', border: 'none', color: '#fff', fontSize: 24, cursor: 'pointer' }} onClick={() => setDocView(null)}>✕</button>
          <div onClick={e => e.stopPropagation()} style={{ width: '90%', height: '90%', background: '#111', borderRadius: 8, overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            {docView.type === 'application/pdf' ? (
              <iframe src={docView.url} style={{ width: '100%', height: '100%', border: 'none' }} title="Document Viewer" />
            ) : (
              <img src={docView.url} alt="Document" style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }} />
            )}
          </div>
        </div>
      )}

      {/* Nav */}
      <nav style={navStyle}>
        <div style={navInner}>
          <div style={{ display: 'flex', gap: 24, alignItems: 'center' }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>KYC Reviewer</span>
            <button onClick={() => setSelected(null)} style={{ ...navBtn, color: !selected ? 'var(--text-primary)' : 'var(--text-secondary)' }}>
              Queue
            </button>
          </div>
          <button id="reviewer-logout" onClick={logout} style={navBtn}>Sign out</button>
        </div>
      </nav>

      <div style={{ maxWidth: 1000, margin: '0 auto', padding: '32px 16px' }}>
        {!selected ? (
          /* === QUEUE VIEW === */
          <div>
            {/* Metrics */}
            <div style={{ display: 'flex', gap: 32, marginBottom: 32, background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: '16px 24px' }}>
              <div>
                <p style={metricLabel}>Queue Size</p>
                <p style={metricValue}>{metrics.queue_size}</p>
              </div>
              <div style={{ width: 1, background: 'var(--border)' }} />
              <div>
                <p style={metricLabel}>Average Wait</p>
                <p style={metricValue}>{metrics.average_wait_hours.toFixed(1)}h</p>
              </div>
              <div style={{ width: 1, background: 'var(--border)' }} />
              <div>
                <p style={metricLabel}>7-day Approval Rate</p>
                <p style={metricValue}>
                  {metrics.approval_rate_7d > 0 ? `${(metrics.approval_rate_7d * 100).toFixed(0)}%` : '—'}
                </p>
              </div>
            </div>

            {/* Table */}
            <div style={{ background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', overflow: 'hidden' }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1.5fr 1.5fr 1fr 1fr 1fr auto', gap: 16, padding: '12px 16px', borderBottom: '1px solid var(--border)', background: 'var(--bg-input)' }}>
                <span style={thStyle}>Merchant Name</span>
                <span style={thStyle}>Business Name</span>
                <span style={thStyle}>Status</span>
                <span style={thStyle}>Submitted</span>
                <span style={thStyle}>Wait Time</span>
                <span style={thStyle}>Action</span>
              </div>
              
              {queue.length === 0 ? (
                <div style={{ padding: 32, textAlign: 'center' }}>
                   <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>No submissions waiting. All caught up.</p>
                </div>
              ) : (
                queue.map((q) => {
                  const waitTimeMs = Date.now() - new Date(q.created_at).getTime();
                  const waitHours = Math.floor(waitTimeMs / (1000 * 60 * 60));
                  const waitMins = Math.floor((waitTimeMs % (1000 * 60 * 60)) / (1000 * 60));
                  
                  return (
                    <div key={q.submission_id} style={{ 
                      display: 'grid', gridTemplateColumns: '1.5fr 1.5fr 1fr 1fr 1fr auto', gap: 16, padding: '16px', borderBottom: '1px solid var(--border)', alignItems: 'center',
                      borderLeft: q.at_risk ? '3px solid var(--danger)' : '3px solid transparent'
                    }}>
                      <span style={{ fontSize: 13, color: 'var(--text-primary)' }}>{q.personal_details?.full_name || '—'}</span>
                      <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{q.business_details?.business_name || '—'}</span>
                      <span style={{ fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase' }}>{q.state}</span>
                      <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{new Date(q.created_at).toLocaleDateString()}</span>
                      <span style={{ fontSize: 13, color: q.at_risk ? 'var(--danger)' : 'var(--text-secondary)' }}>
                        {waitHours > 0 ? `${waitHours}h ` : ''}{waitMins}m
                      </span>
                      <button onClick={() => handleReviewClick(q.submission_id)} style={reviewBtnStyle}>Review →</button>
                    </div>
                  );
                })
              )}
            </div>

            {/* Pagination */}
            {total > limit && (
              <div style={{ display: 'flex', justifyContent: 'center', gap: 16, marginTop: 24 }}>
                <button 
                  onClick={() => setPage(p => Math.max(0, p - 1))} 
                  disabled={page === 0}
                  style={pageBtnStyle}
                >Previous</button>
                <span style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'flex', alignItems: 'center' }}>
                  Page {page + 1} of {Math.ceil(total / limit)}
                </span>
                <button 
                  onClick={() => setPage(p => p + 1)} 
                  disabled={(page + 1) * limit >= total}
                  style={pageBtnStyle}
                >Next</button>
              </div>
            )}
          </div>
        ) : (
          /* === DETAIL VIEW === */
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 32, alignItems: 'start' }}>
            
            {/* Left Col: Details (scrollable implicitly by window) */}
            <div style={{ background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: 32 }}>
              <button 
                onClick={() => setSelected(null)} 
                style={{ ...navBtn, padding: 0, marginBottom: 24, fontSize: 13, display: 'inline-flex', alignItems: 'center', gap: 4 }}
              >
                ← Back to Queue
              </button>

              <Section title="Personal details">
                <DataRow label="Name" value={selected.personal_details?.full_name} />
                <DataRow label="Email" value={selected.personal_details?.email} />
                <DataRow label="Phone" value={selected.personal_details?.phone} />
              </Section>
              
              <Section title="Business details">
                <DataRow label="Business Name" value={selected.business_details?.business_name} />
                <DataRow label="Business Type" value={selected.business_details?.business_type} />
                <DataRow label="Expected Volume" value={`$${selected.business_details?.expected_monthly_volume?.toLocaleString()}`} />
              </Section>

              <Section title="Documents" noBorder>
                {selected.documents?.map((doc) => (
                  <div key={doc.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0' }}>
                    <div>
                      <span style={{ fontSize: 13, color: 'var(--text-primary)', textTransform: 'capitalize', display: 'block', marginBottom: 2 }}>
                        {doc.file_type.replace('_', ' ')}
                      </span>
                      <span style={{ fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)' }}>{doc.original_name}</span>
                    </div>
                    <button 
                      onClick={() => setDocView({ url: `/api/v1/uploads/${doc.storage_key}`, type: doc.mime_type })}
                      style={{...pageBtnStyle, padding: '6px 12px', fontSize: 12}}
                    >
                      View Document
                    </button>
                  </div>
                ))}
              </Section>
            </div>

            {/* Right Col: Action Panel (sticky) */}
            <div style={{ position: 'sticky', top: 32 }}>
               <div style={{ background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', padding: 24 }}>
                  <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 16 }}>
                    State: {selected.state.replace(/_/g, ' ')}
                  </p>
                  
                  {selected.state === 'draft' || selected.state === 'submitted' ? (
                     <div>
                       <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 24 }}>
                         Received {Math.round((Date.now() - new Date(selected.updated_at).getTime()) / (1000 * 60 * 60))} hours ago
                       </p>
                       <button
                         onClick={() => handleTransition('under_review')}
                         disabled={transitioning}
                         style={{ ...actionBtnStyle, background: 'var(--accent)', color: '#000', width: '100%' }}
                       >
                         {transitioning ? 'Starting...' : 'Start Review'}
                       </button>
                     </div>
                  ) : selected.state === 'under_review' ? (
                     <div>
                        <div style={{ marginBottom: 16 }}>
                          <label style={{ display: 'block', fontSize: 13, color: 'var(--text-secondary)', marginBottom: 8 }}>Reviewer Note</label>
                          <textarea
                            value={note}
                            onChange={(e) => setNote(e.target.value)}
                            placeholder="Add reason if rejecting or requesting info..."
                            style={{ ...inputStyle, minHeight: 80, resize: 'vertical', borderColor: noteError ? 'var(--danger)' : 'var(--border)' }}
                          />
                          {noteError && <p style={{ fontSize: 12, color: 'var(--danger)', marginTop: 4 }}>{noteError}</p>}
                        </div>
                        
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                          <button
                            onClick={() => handleTransition('approved')}
                            disabled={transitioning}
                            style={{ ...actionBtnStyle, background: 'var(--success)', color: '#000' }}
                          >
                            Approve
                          </button>
                          <button
                            onClick={() => handleTransition('more_info_requested')}
                            disabled={transitioning}
                            style={{ ...actionBtnStyle, background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-primary)' }}
                          >
                            Request More Info
                          </button>
                          <button
                            onClick={() => handleTransition('rejected')}
                            disabled={transitioning}
                            style={{ ...actionBtnStyle, background: 'transparent', border: '1px solid var(--danger)', color: 'var(--danger)' }}
                          >
                            Reject
                          </button>
                        </div>
                     </div>
                  ) : (
                     <div>
                        <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 16 }}>
                          Reviewed previously
                        </p>
                        {selected.reviewer_note && (
                           <div style={{ padding: 12, background: 'var(--bg-input)', borderRadius: 4, marginBottom: 24 }}>
                             <p style={{ fontSize: 11, color: 'var(--text-tertiary)', textTransform: 'uppercase', marginBottom: 4 }}>Your note:</p>
                             <p style={{ fontSize: 13, color: 'var(--text-primary)' }}>"{selected.reviewer_note}"</p>
                           </div>
                        )}
                        <button onClick={() => setSelected(null)} style={{ ...navBtn, padding: 0 }}>← Back to Queue</button>
                     </div>
                  )}
               </div>
            </div>
            
          </div>
        )}
      </div>
    </div>
  );
}

function Section({ title, children, noBorder }) {
  return (
    <div style={{ paddingBottom: noBorder ? 0 : 24, marginBottom: noBorder ? 0 : 24, borderBottom: noBorder ? 'none' : '1px solid var(--border)' }}>
      <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 16 }}>
        {title}
      </p>
      {children}
    </div>
  );
}

function DataRow({ label, value }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0' }}>
      <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ fontSize: 13, color: 'var(--text-primary)' }}>{value || '—'}</span>
    </div>
  );
}

const navStyle = { borderBottom: '1px solid var(--border)', background: 'var(--bg-card)' };
const navInner = { maxWidth: 1000, margin: '0 auto', padding: '12px 16px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' };
const navBtn = { background: 'transparent', border: 'none', color: 'var(--text-secondary)', fontSize: 13, cursor: 'pointer', fontFamily: 'var(--font-body)', transition: 'color 150ms ease' };
const metricLabel = { fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', marginBottom: 8 };
const metricValue = { fontSize: 24, fontWeight: 500, color: 'var(--text-primary)' };
const thStyle = { fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em' };
const reviewBtnStyle = { background: 'transparent', border: '1px solid var(--border)', borderRadius: 'var(--radius)', color: 'var(--text-primary)', fontSize: 12, padding: '6px 12px', cursor: 'pointer', fontFamily: 'var(--font-mono)' };
const pageBtnStyle = { background: 'var(--bg-card)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', color: 'var(--text-primary)', fontSize: 13, padding: '8px 16px', cursor: 'pointer', fontFamily: 'var(--font-body)' };
const actionBtnStyle = { padding: '10px 16px', borderRadius: 'var(--radius)', fontSize: 14, fontWeight: 500, cursor: 'pointer', border: 'none', fontFamily: 'var(--font-body)', transition: 'opacity 150ms ease' };
const inputStyle = { width: '100%', padding: '10px 12px', background: 'var(--bg-input)', border: '1px solid var(--border)', borderRadius: 'var(--radius)', color: 'var(--text-primary)', fontSize: 14, fontFamily: 'var(--font-body)', outline: 'none' };
