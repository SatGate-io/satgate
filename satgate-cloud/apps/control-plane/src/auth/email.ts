/**
 * Email sending for magic links
 */

import * as nodemailer from 'nodemailer';
import { logger } from '@satgate/common';

// Create transporter (configure via env vars)
const transporter = nodemailer.createTransport({
  host: process.env.SMTP_HOST || 'smtp.example.com',
  port: parseInt(process.env.SMTP_PORT || '587', 10),
  secure: process.env.SMTP_SECURE === 'true',
  auth: {
    user: process.env.SMTP_USER,
    pass: process.env.SMTP_PASS,
  },
});

/**
 * Send magic link email
 */
export async function sendMagicLinkEmail(
  to: string,
  magicLink: string
): Promise<void> {
  const fromEmail = process.env.EMAIL_FROM || 'noreply@satgate.io';
  const fromName = process.env.EMAIL_FROM_NAME || 'SatGate';
  
  const subject = 'Sign in to SatGate';
  
  const html = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sign in to SatGate</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="text-align: center; margin-bottom: 30px;">
    <h1 style="color: #7c3aed; margin: 0;">⚡ SatGate</h1>
  </div>
  
  <p>Click the button below to sign in to your SatGate account:</p>
  
  <div style="text-align: center; margin: 30px 0;">
    <a href="${magicLink}" 
       style="display: inline-block; background: #7c3aed; color: white; padding: 14px 28px; text-decoration: none; border-radius: 8px; font-weight: 600;">
      Sign In
    </a>
  </div>
  
  <p style="color: #666; font-size: 14px;">
    This link will expire in 10 minutes.
  </p>
  
  <p style="color: #666; font-size: 14px;">
    If you didn't request this email, you can safely ignore it.
  </p>
  
  <hr style="border: none; border-top: 1px solid #eee; margin: 30px 0;">
  
  <p style="color: #999; font-size: 12px;">
    If the button doesn't work, copy and paste this link into your browser:<br>
    <a href="${magicLink}" style="color: #7c3aed; word-break: break-all;">${magicLink}</a>
  </p>
  
  <p style="color: #999; font-size: 12px; margin-top: 20px;">
    SatGate — L402 Gateway for the Lightning Economy
  </p>
</body>
</html>
`;

  const text = `
Sign in to SatGate

Click this link to sign in:
${magicLink}

This link will expire in 10 minutes.

If you didn't request this email, you can safely ignore it.
`;

  // In development, just log the link
  if (process.env.NODE_ENV === 'development' || !process.env.SMTP_HOST) {
    logger.info('Magic link (dev mode)', { to, magicLink });
    return;
  }

  await transporter.sendMail({
    from: `"${fromName}" <${fromEmail}>`,
    to,
    subject,
    text,
    html,
  });
}

