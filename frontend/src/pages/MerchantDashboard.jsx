import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { getMySubmission, saveDraft, submitKYC } from '../api/client';
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
  more_info_requested: { label: 'MORE INFO REQUESTED', color: 'var(--danger)' },
};

const STEPS = ['Personal', 'Business', 'Documents'];

export default function MerchantDashboard() {
  const [step, setStep] = useState(0);
  const [submission, setSubmission] = useState(null);
  const [personalDetails, setPersonalDetails] = useState({});
  const [businessDetails, setBusinessDetails] = useState({});
  const [files, setFiles] = useState({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const toast = useToast();

  const fetchSubmission = useCallback(async () => {
    try {
      const res = await getMySubmission();
      setSubmission(res.data);
      if (res.data.personal_details) setPersonalDetails(res.data.personal_details);
      if (res.data.business_details) setBusinessDetails(res.data.business_details);
    } catch {
      // No submission
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchSubmission(); }, [fetchSubmission]);

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
      await fetchSubmission();
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
      await fetchSubmission();
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

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-base)' }}>
        <p style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', fontSize: 13 }}>Loading…</p>
      </div>
    );
  }

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-base)' }}>
      {/* Nav */}
      <nav style={navStyle}>
        <div style={navInner}>
          <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-primary)' }}>KYC</span>
          <button id="merchant-logout" onClick={logout} style={navBtn}>Sign out</button>
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
          </div>
        ) : (
          /* === STATUS VIEW === */
          <div>
            {/* Status badge */}
            <div style={{ textAlign: 'center', marginBottom: 32 }}>
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

            {/* Reviewer note */}
            {submission?.reviewer_note && (submission.state === 'rejected' || submission.state === 'more_info_requested') && (
              <div style={noteBoxStyle}>
                <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', marginBottom: 4, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                  Reviewer note
                </p>
                <p style={{ fontSize: 13, color: 'var(--text-primary)' }}>{submission.reviewer_note}</p>
              </div>
            )}

            {/* Submission data */}
            <div style={cardStyle}>
              {submission?.personal_details && (
                <Section title="Personal details">
                  <DataRow label="Name" value={submission.personal_details.full_name} />
                  <DataRow label="Email" value={submission.personal_details.email} />
                  <DataRow label="Phone" value={submission.personal_details.phone} />
                </Section>
              )}
              {submission?.business_details && (
                <Section title="Business details">
                  <DataRow label="Name" value={submission.business_details.business_name} />
                  <DataRow label="Type" value={submission.business_details.business_type} />
                  <DataRow label="Volume" value={`₹${submission.business_details.expected_monthly_volume?.toLocaleString()}`} />
                </Section>
              )}
              {submission?.documents?.length > 0 && (
                <Section title="Documents" noBorder>
                  {submission.documents.map((doc) => (
                    <div key={doc.id} style={{ display: 'flex', justifyContent: 'space-between', padding: '6px 0' }}>
                      <span style={{ fontSize: 13, color: 'var(--text-primary)', textTransform: 'capitalize' }}>
                        {doc.file_type.replace('_', ' ')}
                      </span>
                      <span style={{ fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)' }}>
                        {doc.original_name}
                      </span>
                    </div>
                  ))}
                </Section>
              )}
            </div>

            {/* Timeline */}
            <div style={{ marginTop: 24 }}>
              <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 12 }}>
                Timeline
              </p>
              <div style={{ position: 'relative', paddingLeft: 20 }}>
                <div style={{ position: 'absolute', left: 4, top: 4, bottom: 4, width: 1, background: 'var(--border)' }} />
                <TimelineEvent label="Created" time={submission.created_at} />
                {submission.state !== 'draft' && <TimelineEvent label="Submitted" time={submission.updated_at} />}
                {(submission.state === 'approved' || submission.state === 'rejected') && (
                  <TimelineEvent
                    label={submission.state === 'approved' ? 'Approved' : 'Rejected'}
                    time={submission.updated_at}
                    color={submission.state === 'approved' ? 'var(--success)' : 'var(--danger)'}
                  />
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
    <div style={{ paddingBottom: noBorder ? 0 : 16, marginBottom: noBorder ? 0 : 16, borderBottom: noBorder ? 'none' : '1px solid var(--border)' }}>
      <p style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 10 }}>
        {title}
      </p>
      {children}
    </div>
  );
}

function DataRow({ label, value }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', padding: '4px 0' }}>
      <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ fontSize: 13, color: 'var(--text-primary)' }}>{value || '—'}</span>
    </div>
  );
}

function TimelineEvent({ label, time, color }) {
  return (
    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, position: 'relative', paddingBottom: 16 }}>
      <div style={{ width: 9, height: 9, borderRadius: '50%', background: color || 'var(--text-tertiary)', flexShrink: 0, marginTop: 3, position: 'relative', zIndex: 1 }} />
      <div>
        <p style={{ fontSize: 13, color: color || 'var(--text-primary)' }}>{label}</p>
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
