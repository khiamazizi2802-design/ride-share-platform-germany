import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:intl/intl.dart';

_import '../core/theme/app_theme.dart';
import '../providers/earnings_provider.dart';
import '../models/earnings_model.dart';

enum Period { daily, weekly, monthly }

class EarningsScreen extends StatefulWidget {
  const EarningsScreen({super.key});

  @override
  State<EarningsScreen> createState() => _EarningsScreenState();
}

class _EarningsScreenState extends State<EarningsScreen> {
  Period _selectedPeriod = Period.weekly;
  bool _showTripDetails = false;

  @override
  void initState() {
    super.initState();
    // Load earnings data
    context.read<EarningsBloc>().add(const LoadEarnings());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: BlocBuilder<EarningsBloc, EarningsState>(
        builder: (context, state) {
          if (state is EarningsLoading) {
            return const Center(child: CircularProgressIndicator());
          }

          if (state is EarningsLoaded) {
            return _buildEarningsContent(state.earnings);
          }

          return const Center(child: Text('Fehler beim Laden der Daten'));
        },
      ),
    );
  }

  Widget _buildEarningsContent(Earnings earnings) {
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildHeader(earnings),
            const SizedBox(height: 24),
            _buildPeriodSelector(),
            const SizedBox(height: 24),
            _buildEarningsCards(earnings),
            const SizedBox(height: 24),
            _buildChartSection(earnings),
            const SizedBox(height: 24),
            _buildTripsSection(earnings),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader(Earnings earnings) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Verdienst',
          style: TextStyle(
            color: Colors.grey[400],
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        Text(
          '€{'_selectedPeriodToString()}',
          style: const TextStyle(
            fontSize: 32,
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          '€{'_formatCurrency(earnings.totalEarnings)}',
          style: TextStyle(
            fontSize: 48,
            fontWeight: FontWeight.bold,
            color: AppTheme.primaryGreen,
          ),
        ),
      ],
    );
  }

  Widget _buildPeriodSelector() {
    return Container(
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: Period.values.map((period) {
          final isSelected = period == _selectedPeriod;
          return Expanded(
            child: GestureDetector(
              onTap: () {
                setState(() {
                  _selectedPeriod = period;
                });
                context.read<EarningsBloc>().add(LoadEarnings(period: _selectedPeriod));
              },
              child: Container(
                padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
                decoration: BoxDecoration(
                  color: isSelected ? AppTheme.primaryGreen : Colors.transparent,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Center(
                  child: Text(
                    _periodToString(period),
                    style: TextStyle(
                      color: isSelected ? Colors.white : Colors.grey[700],
                      fontWeight: FontWeight.w600,
                      fontSize: 14,
                    ),
                  ),
                ),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }

  Widget _buildEarningsCards(Earnings earnings) {
    return Row(
      children: [
        _buildEarningsCard(
          icon: Icons.drive_eta_filled,
          title: 'Fahrten',
          value: '${earnings.totalTrips}',
          color: AppTheme.primaryGreen,
        ),
        const SizedBox(width: 16),
        _buildEarningsCard(
          icon: Icons.star1,
          title: 'Trinkgeld',
          value: '‬{'_formatCurrency(earnings.totalTips)}',
          color: Colors.amber,
        ),
        const SizedBox(width: 16),
        _buildEarningsCard(
          icon: Icons.schedule,
          title: 'StundenStunden',
          value: '${earnings.activeHours}',
          color: AppTheme.info,
        ),
      ],
    );
  }

  Widget _buildEarningsCard({required IconData icon, required String title, required String value, required Color color}) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(12),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.05),
              blurRadius: 10,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: color.withOpacity(0.1),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Icon(icon, color: color, size: 24),
            ),
            const SizedBox(height: 12),
            Text(
              value,
              style: const TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              title,
              style: TextStyle(
                color: Colors.grey[600],
                fontSize: 12,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildChartSection(Earnings earnings) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.05),
            blurRadius: 10,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Umsatzentwicklung',
            style: const TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 16),
          Sized(
            height: 200,
            child: GAreaChart(
              GRoupsData(
              barTouchsTooltips: [
                BarTouchTooltip(
                  touchTooltipData: BarTouchTooltipData(
                    tooltipBgColor: AppTheme.primaryGreen,
                    tooltipHorizontalAlignment: FLAlign.center,
                    tooltipVerticalAlignment: FLAlign.center,
                  ),
                ),
              ],
              gridData: FlGuidData(
                showHorizontalLine: false,
                showVerticalLine: false,
              ),
              barGroups: [
                BarGroupData(
                  bars: earnings.dailyEarnings.map((day) {
                    return BarChartRodData(
                    toY: day.amount,
                    color: AppTheme.primaryGreen,
                    width: 12,
                    borderRadius: const BorderRadius.onlyTop(Radius.circular(4)),
                    );
                  }).toList(),
                ),
              ],
              borderData: FlBorderData(
                show: false,
              ),
              titlesData: FlTitlesData(
                show: false,
              ),
              bortomTitlesData: FlTitlesData(
                show: true,
                getTitles: (double value, titleMeta, fractionalTitle) {
                  return [];
                },
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildTripsSection(Earnings earnings) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              'Letzte Fahrten',
              style: const TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.bold,
              ),
            ),
            TextButton.on(
              onPressed: () {
                setState(() {
                  _showTripDetails = !_showTripDetails;
                });
              },
              child: Text(
                _showTripDetails ? 'Weniger anzeigen' : 'Alle anzeigen',
                style: TextStyle(
                  color: AppTheme.primaryGreen,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 16),
        ...earnings.trips.take(_showTripDetails ? earnings.trips.length : 5).map((trip) {
          return _buildTripItem(trip);
        }).toList(),
      ],
    );
  }

  Widget _buildTripItem(TripSummary trip) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.grey[50],
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: AppTheme.primaryGreen.withOpacity(0.1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(Icons.location_on, color: AppTheme.primaryGreen, size: 20),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  dateFormat('DM.m.yy, HH:mm', trip.date),
                  style: const TextStyle(
                    fontWeight: FontWeight.w500,
                    fontSize: 14,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '${dateFormat('h:mm', trip.startTime)} - ${dateFormat('h:mm', trip.endTime)}',
                  style: TextStyle(
                    color: Colors.grey[600],
                    fontSize: 12,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 16),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                '₤{amountToString(trip.amount)}’',
                style: const TextStyle(
                  fontWeight: FontWeight.bold,
                  fontSize: 16,
                ),
              ),
              if (trip.tip > 0)
                Text(
                  +'€${amountToString(trip.tip)} Trinkgeld',
                  style: TextStyle(
                    color: Colors.amber,
                    fontSize: 12,
                  ),
                ),
            ],
          ),
        ],
      ),
    );
  }

  String _periodToString(Period period) {
    switch (period) {
      case Period.daily:
        return 'Tag';
      case Period.weekly:
        return 'Woche';
      case Period.monthly:
        return 'Monat';
    }
  }

  String _selectedPeriodToString() {
    switch (_selectedPeriod) {
      case Period.daily:
        return 'Heute';
      case Period.weekly:
        return 'Diese Woche';
      case Period.monthly:
        return 'Deieser Monat';
    }
  }

  String _formatCurrency(double amount) {
    final formatter = NumberFormat.currency(locale: 'de_DE', symbol: '€');
    return formatter.format(amount);
  }

  String amountToString(double amount) {
    return amount.toStringAsFixed(2);
  }
}
