import React from 'react';
import { Car, Shield, Smartphone, Globe, ArrowRight, CheckCircle } from 'lucide-react';

const LandingPage: React.FC = () => {
  return (
    <div className="min-h-screen bg-white font-sans text-slate-900">
      {/* Navbar */}
      <nav className="border-b border-slate-100 py-4 px-6 md:px-12 flex justify-between items-center sticky top-0 bg-white/80 backdrop-blur-md z-50">
        <div className="flex items-center gap-2">
          <div className="w-10 h-10 bg-green-500 rounded-xl flex items-center justify-center text-white font-bold text-2xl shadow-lg shadow-green-200">
            G
          </div>
          <span className="text-2xl font-black tracking-tight text-slate-900">GREEN</span>
        </div>
        <div className="hidden md:flex items-center gap-8 font-medium">
          <a href="#rider" className="hover:text-green-600 transition-colors">Rider</a>
          <a href="#driver" className="hover:text-green-600 transition-colors">Driver</a>
          <a href="#safety" className="hover:text-green-600 transition-colors">Safety</a>
          <button className="bg-slate-900 text-white px-6 py-2.5 rounded-full hover:bg-slate-800 transition-all active:scale-95 shadow-lg shadow-slate-200">
            Sign In
          </button>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="relative pt-20 pb-32 px-6 md:px-12 overflow-hidden">
        <div className="max-w-7xl mx-auto grid md:grid-cols-2 gap-16 items-center">
          <div className="relative z-10">
            <div className="inline-flex items-center gap-2 bg-green-50 text-green-700 px-4 py-2 rounded-full text-sm font-bold mb-6 border border-green-100">
              <Shield size={16} /> Fully German BFSG & PBefG Compliant
            </div>
            <h1 className="text-6xl md:text-7xl font-black leading-[1.1] mb-8 tracking-tight">
              Move <span className="text-green-500">Green</span>.<br />Move Smarter.
            </h1>
            <p className="text-xl text-slate-500 mb-10 max-w-lg leading-relaxed">
              The first world-class ride-sharing platform built specifically for the German market. Professional, reliable, and 100% compliant.
            </p>
            <div className="flex flex-col sm:flex-row gap-4">
              <button className="bg-green-500 text-white px-10 py-4 rounded-2xl font-bold text-lg hover:bg-green-600 transition-all flex items-center justify-center gap-2 shadow-xl shadow-green-200 active:scale-95 group">
                Sign Up to Ride <ArrowRight className="group-hover:translate-x-1 transition-transform" />
              </button>
              <button className="bg-white text-slate-900 border-2 border-slate-100 px-10 py-4 rounded-2xl font-bold text-lg hover:border-slate-300 transition-all active:scale-95">
                Drive with Green
              </button>
            </div>
          </div>
          <div className="relative">
            <div className="absolute -top-24 -right-24 w-96 h-96 bg-green-400/10 rounded-full blur-3xl opacity-50 animate-pulse"></div>
            <div className="absolute -bottom-24 -left-24 w-96 h-96 bg-blue-400/10 rounded-full blur-3xl opacity-50"></div>
            <div className="bg-slate-50 rounded-[3rem] p-8 border border-slate-100 shadow-2xl relative overflow-hidden aspect-square flex items-center justify-center">
               <Car size={300} className="text-slate-200 absolute -bottom-10 -right-10 rotate-12" />
               <div className="relative z-10 text-center">
                 <div className="w-24 h-24 bg-green-500 rounded-3xl mx-auto mb-6 flex items-center justify-center text-white shadow-2xl shadow-green-200">
                   <Smartphone size={48} />
                 </div>
                 <h3 className="text-2xl font-bold mb-2">Download the App</h3>
                 <p className="text-slate-500">Coming soon to App Store & Play Store</p>
               </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="py-32 bg-slate-50" id="safety">
        <div className="max-w-7xl mx-auto px-6 md:px-12">
          <div className="text-center max-w-3xl mx-auto mb-20">
            <h2 className="text-4xl md:text-5xl font-black mb-6">Designed for Excellence</h2>
            <p className="text-xl text-slate-500">We've combined world-class UI/UX with German engineering standards to create a seamless experience.</p>
          </div>
          <div className="grid md:grid-cols-3 gap-8">
            {[
              { icon: <Shield className="text-green-500" />, title: "Safety First", desc: "SOS buttons, ride sharing, and fully verified professional drivers." },
              { icon: <Globe className="text-green-500" />, title: "Fully Compliant", desc: "Adhering to PBefG and local city regulations in Berlin, Munich, & more." },
              { icon: <CheckCircle className="text-green-500" />, title: "Transparent Pricing", desc: "No hidden fees. Precise estimates before you book, every time." }
            ].map((f, i) => (
              <div key={i} className="bg-white p-10 rounded-[2.5rem] border border-slate-100 shadow-sm hover:shadow-xl hover:-translate-y-2 transition-all duration-300">
                <div className="w-16 h-16 bg-green-50 rounded-2xl flex items-center justify-center mb-8">
                  {React.cloneElement(f.icon as React.ReactElement, { size: 32 })}
                </div>
                <h3 className="text-2xl font-bold mb-4">{f.title}</h3>
                <p className="text-slate-500 leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-20 px-6 md:px-12 border-t border-slate-100">
        <div className="max-w-7xl mx-auto flex flex-col md:row justify-between items-center gap-12">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-green-500 rounded-lg flex items-center justify-center text-white font-bold text-lg">G</div>
            <span className="text-xl font-black">GREEN</span>
          </div>
          <div className="flex gap-8 text-slate-500 font-medium">
            <a href="#" className="hover:text-green-600 transition-colors">Legal</a>
            <a href="#" className="hover:text-green-600 transition-colors">Privacy</a>
            <a href="#" className="hover:text-green-600 transition-colors">Terms</a>
            <a href="#" className="hover:text-green-600 transition-colors">Contact</a>
          </div>
          <p className="text-slate-400 text-sm">© 2026 Green Ride-Sharing Platform. Built with Twin.</p>
        </div>
      </footer>
    </div>
  );
};

export default LandingPage;
