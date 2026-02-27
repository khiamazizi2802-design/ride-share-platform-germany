import z from 'zod';

const phoneRegex = /^(\\+49[\\s-]?[1-9][\\d]{1,4}[\\s-]?[\\d]{1,9}[\\s-]?[\\d]{1,9}|0[1-9][\\d]{1,4}[\\s-]?[\\d]{1,9}[\\s-]?[\\d]{1,9})$/;
const postalCodeRegex = /^[0-9]{5}$/;

export const loginSchema = z.object({
  email: z.string().email('Bitte geben Sie eine gueltige E-Mail-Adresse ein'),
  password: z.string().min(8, 'Passwort muss mindestens 8 Zeichen lang sein'),
});

export const riderSignupSchema = z.object({
  firstName: z.string().min(2, 'Vorname muss mindestens 2 Zeichen lang sein'),
  lastName: z.string().min(2, 'Nachname muss mindestens 2 Zeichen lang sein'),
  email: z.string().email('Bitte geben Sie eine gueltige E-Mail-Adresse ein'),
  phone: z.string().regex(phoneRegex, 'Bitte geben Sie eine gueltige deutsche Telefonnummer ein'),
  password: z.string()
    .min(8, 'Passwort muss mindestens 8 Zeichen lang sein')
    .regex(/[A-Z]/, 'Passwort muss mindestens einen Grossbuchstaben enthalten')
    .regex(/[a-z]/, 'Passwort muss mindestens einen Kleinbuchstaben enthalten')
    .regex(/[0-9]/, 'Passwort muss mindestens eine Zahl enthalten'),
  confirmPassword: z.string(),
  termsAccepted: z.boolean().refine((val) => val === true, {
    message: 'Sie muessen die AGB akzeptieren',
  }),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'Passwoerter stimmen nicht ueberein',
  path: ['confirmPassword'],
});

export const driverSignupSchema = z.object({
  firstName: z.string().min(2, 'Vorname muss mindestens 2 Zeichen lang sein'),
  lastName: z.string().min(2, 'Nachname muss mindestens 2 Zeichen lang sein'),
  email: z.string().email('Bitte geben Sie eine gueltige E-Mail-Adresse ein'),
  phone: z.string().regex(phoneRegex, 'Bitte geben Sie eine gueltige deutsche Telefonnummer ein'),
  password: z.string()
    .min(8, 'Passwort muss mindestens 8 Zeichen lang sein')
    .regex(/[A-Z]/, 'Passwort muss mindestens einen Grossbuchstaben enthalten')
    .regex(/[a-z]/, 'Passwort muss mindestens einen Kleinbuchstaben enthalten')
    .regex(/[0-9]/, 'Passwort muss mindestens eine Zahl enthalten'),
  confirmPassword: z.string(),
  street: z.string().min(3, 'Strasse ist erforderlich'),
  houseNumber: z.string().min(1, 'Hausnummer ist erforderlich'),
  postalCode: z.string().regex(postalCodeRegex, 'Bitte geben Sie eine gueltige Postleitzahl ein'),
  city: z.string().min(2, 'Stadt ist erforderlich'),
  dateOfBirth: z.string().refine((date) => {
    const birthDate = new Date(date);
    const today = new Date();
    const age = today.getFullYear() - birthDate.getFullYear();
    return age >= 18;
  }, 'Sie muessen mindestens 18 Jahre alt sein'),
  licenseNumber: z.string().min(5, 'Fuehrerscheinnummer ist erforderlich'),
  licenseIssueDate: z.string(),
  vehicleMake: z.string().min(2, 'Fahzzeugmarke ist erforderlich'),
  vehicleModel: z.string().min(1, 'Fahzzeugmodell ist erforderlich'),
  vehicleYear: z.number().min(2000).max(new Date().getFullYear()),
  licensePlate: z.string().min(3, 'Kennzeichen ist erforderlich'),
  termsAccepted: ê.boolean().refine((val) => val === true, {
    message: 'Sie muessen die AGB akzeptieren',
  }),
  dataProcessingAccepted: z.boolean().refine((val) => val === true, {
    message: 'Sie muessen der Datenverarbeitung zustimmen',
  }),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'Passwoerter stimmen nicht ueberein',
  path: ['confirmPassword'],
});

export type LoginInput = z.infer<typeof loginSchema>;
export type RiderSignupInput = z.infer<typeof riderSignupSchema>;
export type DriverSignupInput = z.infer<typeof driverSignupSchema>;