import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { getMySubmission, saveDraft, submitKYC, getNotifications } from '../api/client';
import { useToast } from '../App';
import Step1Personal from '../components/KYCForm/Step1Personal';
import Step2Business from '../components/KYCForm/Step2Business';
import Step3Documents from '../components/KYCForm/Step3Documents';

const STATE_CONFIG = {
  draft: { label: 'DRAFT', color: 'var(--text-secondary)' },
  submitted: { label: 'SUBMITTED', color: 'var(--text-secondary)' },
  under_review: { label: 'UNDER REVIEW', color: 'var(--text-secondary)' },
  approved: { label: 'APPROVED', color: 'var(--success)' },
  rejected: { label: 'REJECTED', color: 'var(--danger)' },
  more_info_requested: { label: 'MORE INFO REQUESTED', color: 'var(--warning)' },
};

const STEPS = ['Personal', 'Business', 'Documents'];

export default function MerchantDashboard() {
  const [step, setStep] = useState(0);
  const [submission, setSubmission] = useState(null);
  const [notifications, setNotifications] = useState([]);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [personalDetails, setPersonalDetails] = useState({});
  const [businessDetails, setBusinessDetails] = useState({});
  const [files, setFiles] = useState({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const toast = useToast();

  const fetchData = useCallback(async () => {
    try {
      const [subRes, notifRes] = await Promise.all([
        getMySubmission(),
        getNotifications().catch(() => ({ data: { notifications: [] } }))
      ]);
      setSubmission(subRes.data);
      setNotifications(notifRes.data.notifications || []);
      if (subRes.data.personal_details) setPersonalDetails(subRes.data.personal_details);
      if (subRes.data.business_details) setBusinessDetails(subRes.data.business_details);
    } catch {
      // No submission yet
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  // Handle escape to close drawer
  useEffect(() => {
    const handleEscape = (e) => {
      if (e.key === 'Escape') setIsDrawerOpen(false);
    };
    window.addEventListener('keydown', handleEscape);
    return () => window.removeEventListener('keydown', handleEscape);
  }, []);

  const handleSaveDraft = async () => {
    setSaving(true);
    try {
      const formData = new FormData();
      if (Object.keys(personalDetails).length > 0)
        formData.append('personal_details', JSON.stringify(personalDetails));
      if (Object.keys(businessDetails).length > 0)
        formData.append('business_details', JSON.stringify(businessDetails));
      Object.entries(files).forEach(([key, file]) => { if (file) formData.append(key, file); });
      await saveDraft(formData);
      toast('Draft saved', 'success');
      await fetchData();
    } catch (err) {
      toast(err.response?.data?.error || 'Failed to save draft');
    } finally {
      setSaving(false);
    }
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      const formData = new FormData();
      formData.append('personal_details', JSON.stringify(personalDetails));
      formData.append('business_details', JSON.stringify(businessDetails));
      Object.entries(files).forEach(([key, file]) => { if (file) formData.append(key, file); });
      await saveDraft(formData);
      await submitKYC();
      toast('Application submitted for review', 'success');
      await fetchData();
    } catch (err) {
      toast(err.response?.data?.error || 'Failed to submit');
    } finally {
      setSubmitting(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('role');
    navigate('/login');
  };

  const canEdit = !submission || submission.state === 'draft' || submission.state === 'more_info_requested';
  
  // Mark all notifications as read when drawer opens (simplified logic for now)
  const unreadCount = isDrawerOpen ? 0 : (notifications?.length > 0 ? 1 : 0); // Need actual read state if implemented in backend

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-base)' }}>
        <p style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', fontSize: 13 }}>Loading…</p>
      </div>
    );
  }

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-base)', position: 'relative', overflow: isDrawerOpen ? 'hidden' : 'auto' }}>
      
      {/* Drawer Overlay */}
      {isDrawerOpen && (
        <div 
          onClick={() => setIsDrawerOpen(false)}
          style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', zIndex: 40 }}
        />
      )}

      {/* Notification Drawer */}
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0, width: '100%', maxWidth: 360,
        background: 'var(--bg-card)', borderLeft: '1px solid var(--border)',
        transform: isDrawerOpen ? 'translateX(0)' : 'translateX(100%)',
        transition: 'transform 250ms ease', zIndex: 50,
        display: 'flex', flexDirection: 'column'
      }}>
        <div style={{ padding: '20px', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ fontSize: 16, fontWeight: 600, color: 'var(--text-primary)' }}>Notifications</h2>
          <button onClick={() => setIsDrawerOpen(false)} style={navBtn}>✕</button>
        </div>
        <div style={{ padding: 20, overflowY: 'auto', flex: 1, display: 'flex', flexDirection: 'column', gap: 16 }}>
          {notifications && notifications.length > 0 ? (
            notifications.map((n) => (
              <div key={n.id} style={{ padding: 16, background: 'var(--bg-input)', borderRadius: 'var(--radius)', border: '1px solid var(--border)' }}>
                <p style={{ fontSize: 12, color: 'var(--text-tertiary)', fontFamily: 'var(--font-mono)', marginBottom: 4 }}>
                  {new Date(n.created_at).toLocaleString()}
                </p>
                <p style={{ fontSize: 14, color: 'var(--text-primary)' }}>
                  State changed to <strong>{n.event_type}</strong>
                </p>
                {n.payload?.note && (
                  <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginTop: 8, padding: '8px 12px', background: 'var(--bg-card)', borderRadius: 4 }}>
                    "{n.payload.note}"
                  </p>
                )}
              </div>
            ))
          ) : (
            <p style={{ fontSize: 13, color: 'var(--text-tertiary)' }}>No notifications yet.</p>
          )}
        </div>
      </div>

      {/* Nav */}
      <nav style={navStyle}>
        <div style={navInner}>
          <div style={{ display: 'flex', gap: 24, alignItems: 'center' }}>
            <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>KYC Flow</span>
            {canEdit && <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>KYC Form</span>}
            {!canEdit && <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Status</span>}
          </div>
          <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
            <button onClick={() => setIsDrawerOpen(true)} style={{ ...navBtn, position: 'relative' }}>
              🔔 {unreadCount > 0 && <div style={badgeStyle} />}
            </button>
            <button onClick={logout} style={navBtn}>Sign out</button>
          </div>
        </div>
      </nav>

      <div style={{ maxWidth: 600, margin: '0 auto', padding: '32px 16px' }}>
        {canEdit ? (
          /* === FORM VIEW === */
          <div>
            {/* Reviewer note banner */}
            {submission?.reviewer_note && submission.state === 'more_info_requested' && (
              <div style={noteBoxStyle}>
                <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', marginBottom: 4, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Reviewer note
                </p>
                <p style={{ fontSize: 13, color: 'var(--text-primary)' }}>{submission.reviewer_note}</p>
              </div>
            )}

            {/* Step indicator */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 32 }}>
              {STEPS.map((s, i) => (
                <div key={s} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <button
                    onClick={() => setStep(i)}
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: '50%',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 12,
                      fontFamily: 'var(--font-mono)',
                      fontWeight: 500,
                      border: 'none',
                      cursor: 'pointer',
                      transition: 'all 150ms ease',
                      background: i === step ? 'var(--accent)' : i < step ? 'var(--text-secondary)' : 'var(--bg-input)',
                      color: i === step ? '#0A0A0A' : i < step ? '#0A0A0A' : 'var(--text-tertiary)',
                    }}
                  >
                    {i < step ? '✓' : i + 1}
                  </button>
                  <span style={{ fontSize: 13, color: i === step ? 'var(--text-primary)' : 'var(--text-tertiary)' }}>{s}</span>
                  {i < STEPS.length - 1 && (
                    <div style={{ width: 24, height: 1, background: i < step ? 'var(--text-secondary)' : 'var(--border)' }} />
                  )}
                </div>
              ))}
            </div>

            {/* Form card */}
            <div style={cardStyle}>
              {step === 0 && <Step1Personal data={personalDetails} onChange={setPersonalDetails} onNext={() => setStep(1)} />}
              {step === 1 && <Step2Business data={businessDetails} onChange={setBusinessDetails} onNext={() => setStep(2)} onBack={() => setStep(0)} />}
              {step === 2 && <Step3Documents files={files} onChange={setFiles} onBack={() => setStep(1)} onSubmit={handleSubmit} loading={submitting} />}
            </div>

            {/* Save draft */}
            <button id="save-draft-btn" onClick={handleSaveDraft} disabled={saving} style={draftBtnStyle}>
              {saving ? 'Saving…' : 'Save as draft'}
            </button>
            {submission && submission.state === 'more_info_requested' && (
              <button onClick={handleSubmit} disabled={submitting} style={{...draftBtnStyle, marginTop: 24, color: 'var(--accent)'}}>
                {submitting ? 'Resubmitting...' : 'Resubmit KYC'}
              </button>
            )}
          </div>
        ) : (
          /* === STATUS VIEW === */
          <div>
            {/* Status badge */}
            <div style={{ textAlign: 'center', marginBottom: 16 }}>
              <span style={{
                display: 'inline-block',
                padding: '8px 20px',
                border: `1px solid ${STATE_CONFIG[submission.state]?.color || 'var(--border)'}`,
                borderRadius: 'var(--radius)',
                fontFamily: 'var(--font-mono)',
                fontSize: 13,
                fontWeight: 600,
                color: STATE_CONFIG[submission.state]?.color,
                letterSpacing: '0.08em',
              }}>
                {STATE_CONFIG[submission.state]?.label}
              </span>
            </div>
            
            <p style={{ textAlign: 'center', fontSize: 13, color: 'var(--text-secondary)', marginBottom: 32 }}>
              Submitted {Math.round((Date.now() - new Date(submission.updated_at).getTime()) / (1000 * 60 * 60))} hours ago
            </p>

            <div style={{ textAlign: 'center', padding: '16px', background: 'var(--bg-input)', borderRadius: 'var(--radius)', marginBottom: 32 }}>
              {submission.state === 'submitted' || submission.state === 'under_review' ? (
                <p style={{ fontSize: 14, color: 'var(--text-primary)' }}>Your submission is being reviewed. We'll notify you of any updates.</p>
              ) : submission.state === 'approved' ? (
                <p style={{ fontSize: 14, color: 'var(--success)' }}>Your account is approved. You can now start collecting payments.</p>
              ) : submission.state === 'rejected' ? (
                 <div style={{...noteBoxStyle, marginBottom: 0, textAlign: 'left'}}>
                    <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', marginBottom: 4, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                      Reviewer note
                    </p>
                    <p style={{ fontSize: 13, color: 'var(--text-primary)' }}>{submission.reviewer_note}</p>
                  </div>
              ) : null}
            </div>

            {/* Timeline */}
            <div style={{ marginTop: 24 }}>
              <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 12 }}>
                Timeline
              </p>
              <div style={{ position: 'relative', paddingLeft: 20 }}>
                <div style={{ position: 'absolute', left: 4, top: 4, bottom: 4, width: 1, background: 'var(--border)' }} />
                
                {notifications && notifications.length > 0 ? (
                  notifications.map((n, i) => (
                    <TimelineEvent 
                      key={n.id} 
                      label={n.event_type.replace(/_/g, ' ')} 
                      time={n.created_at} 
                      color={n.event_type === 'approved' ? 'var(--success)' : n.event_type === 'rejected' || n.event_type === 'more_info_requested' ? 'var(--danger)' : 'var(--text-primary)'}
                    />
                  ))
                ) : (
                  <>
                    {submission.state !== 'draft' && <TimelineEvent label="Submitted" time={submission.updated_at} />}
                    <TimelineEvent label="Draft" time={submission.created_at} />
                  </>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function TimelineEvent({ label, time, color }) {
  return (
    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, position: 'relative', paddingBottom: 16 }}>
      <div style={{ width: 9, height: 9, borderRadius: '50%', background: color || 'var(--text-tertiary)', flexShrink: 0, marginTop: 3, position: 'relative', zIndex: 1 }} />
      <div>
        <p style={{ fontSize: 13, color: color || 'var(--text-primary)', textTransform: 'capitalize' }}>{label}</p>
        <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', marginTop: 2 }}>
          {time ? new Date(time).toLocaleString() : '—'}
        </p>
      </div>
    </div>
  );
}

const navStyle = {
  borderBottom: '1px solid var(--border)',
  background: 'var(--bg-card)',
};

const navInner = {
  maxWidth: 600,
  margin: '0 auto',
  padding: '12px 16px',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
};

const navBtn = {
  background: 'transparent',
  border: 'none',
  color: 'var(--text-secondary)',
  fontSize: 13,
  cursor: 'pointer',
  fontFamily: 'var(--font-body)',
  transition: 'color 150ms ease',
};

const badgeStyle = {
  position: 'absolute',
  top: 2,
  right: 2,
  width: 6,
  height: 6,
  borderRadius: '50%',
  background: 'var(--danger)',
}

const cardStyle = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius)',
  padding: 24,
};

const noteBoxStyle = {
  background: 'var(--bg-input)',
  border: '1px solid var(--border)',
  borderLeft: '2px solid var(--danger)',
  borderRadius: 'var(--radius)',
  padding: '12px 16px',
  marginBottom: 24,
};

const draftBtnStyle = {
  display: 'block',
  margin: '16px auto 0',
  background: 'transparent',
  border: 'none',
  color: 'var(--text-tertiary)',
  fontSize: 13,
  fontFamily: 'var(--font-mono)',
  cursor: 'pointer',
  transition: 'color 150ms ease',
};
